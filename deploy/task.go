package deploy

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
)

// Task has target ECS information, client of aws-sdk-go, command and timeout seconds.
type Task struct {
	awsECS ecsiface.ECSAPI

	// Name of ECS cluster.
	Cluster string

	// Name of the container for override task definition.
	Name string

	// Name of base task definition for run task.
	BaseTaskDefinition *string

	// TaskDefinition struct to call aws API.
	TaskDefinition *TaskDefinition

	// New image for deploy.
	NewImage *Image

	// Task command which run on ECS.
	Command []*string

	// Wait time when run task.
	// This script monitors ECS task for new task definition to be running after call run task API.
	Timeout time.Duration
}

// NewTask returns a new Task struct, and initialize aws ecs API client.
// Separates imageWithTag into repository and tag, then set NewImage for deploy.
func NewTask(cluster, name, imageWithTag, command string, baseTaskDefinition *string, timeout time.Duration, profile, region string) (*Task, error) {
	if baseTaskDefinition == nil {
		return nil, errors.New("task definition is required")
	}
	awsECS := ecs.New(session.New(), newConfig(profile, region))
	taskDefinition := NewTaskDefinition(profile, region)
	var newImage *Image
	if len(imageWithTag) > 0 {
		var err error
		repository, tag, err := divideImageAndTag(imageWithTag)
		if err != nil {
			return nil, err
		}
		newImage = &Image{
			*repository,
			*tag,
		}
	}
	p := shellwords.NewParser()
	commands, err := p.Parse(command)
	if err != nil {
		return nil, errors.Wrap(err, "Parse error in a task command")
	}
	var cmd []*string
	for _, c := range commands {
		cmd = append(cmd, aws.String(c))
	}

	return &Task{
		awsECS:             awsECS,
		Cluster:            cluster,
		Name:               name,
		BaseTaskDefinition: baseTaskDefinition,
		TaskDefinition:     taskDefinition,
		NewImage:           newImage,
		Command:            cmd,
		Timeout:            timeout,
	}, nil
}

// RunTask calls run-task API.
func (t *Task) RunTask(taskDefinition *ecs.TaskDefinition) ([]*ecs.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), t.Timeout)
	defer cancel()

	containerOverride := &ecs.ContainerOverride{
		Command: t.Command,
		Name:    aws.String(t.Name),
	}

	override := &ecs.TaskOverride{
		ContainerOverrides: []*ecs.ContainerOverride{
			containerOverride,
		},
	}

	params := &ecs.RunTaskInput{
		Cluster:        aws.String(t.Cluster),
		TaskDefinition: taskDefinition.TaskDefinitionArn,
		Overrides:      override,
	}
	resp, err := t.awsECS.RunTaskWithContext(ctx, params)
	if err != nil {
		return nil, err
	}
	if len(resp.Failures) > 0 {
		log.Printf("[ERROR] Run task error: %+v\n", resp.Failures)
		return nil, errors.New(*resp.Failures[0].Reason)
	}
	log.Printf("[INFO] Running tasks: %+v\n", resp.Tasks)

	err = t.waitRunning(ctx, resp.Tasks)
	if err != nil {
		return resp.Tasks, err
	}
	return resp.Tasks, nil
}

// waitRunning waits a task running.
func (t *Task) waitRunning(ctx context.Context, tasks []*ecs.Task) error {
	log.Println("[INFO] Waiting for running task...")

	taskArns := []*string{}
	for _, task := range tasks {
		taskArns = append(taskArns, task.TaskArn)
	}
	errCh := make(chan error, 1)
	done := make(chan struct{}, 1)
	go func() {
		err := t.waitExitTasks(taskArns)
		if err != nil {
			errCh <- err
		}
		close(done)
	}()
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-done:
		log.Println("[INFO] Run task is success")
	case <-ctx.Done():
		return errors.New("process timeout")
	}

	return nil
}

func (t *Task) waitExitTasks(taskArns []*string) error {
retry:
	for {
		time.Sleep(5 * time.Second)

		params := &ecs.DescribeTasksInput{
			Cluster: aws.String(t.Cluster),
			Tasks:   taskArns,
		}
		resp, err := t.awsECS.DescribeTasks(params)
		if err != nil {
			return err
		}

		for _, task := range resp.Tasks {
			if !t.checkTaskStopped(task) {
				continue retry
			}
		}

		for _, task := range resp.Tasks {
			code, result, err := t.checkTaskSucceeded(task)
			if err != nil {
				continue retry
			}
			if !result {
				return errors.Errorf("exit code: %v", code)
			}
		}
		return nil
	}
}

func (t *Task) checkTaskStopped(task *ecs.Task) bool {
	if *task.DesiredStatus != "STOPPED" {
		return false
	}
	return true
}

func (t *Task) checkTaskSucceeded(task *ecs.Task) (int64, bool, error) {
	for _, c := range task.Containers {
		if c.ExitCode == nil {
			return 1, false, errors.New("can not read exit code")
		}
		if *c.ExitCode != int64(0) {
			return *c.ExitCode, false, nil
		}
	}
	return int64(0), true, nil
}
