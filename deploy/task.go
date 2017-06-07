package deploy

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

// Task have image and task definition information
type Task struct {
	image          *Image
	taskDefinition *ecs.TaskDefinition
}

// Image have repository and tag string
type Image struct {
	repository string
	tag        string
}

// TaskDefinition get a current task definition
func (d *Deploy) TaskDefinition(service *ecs.Service) (*ecs.TaskDefinition, error) {
	params := &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(*service.TaskDefinition),
	}
	resp, err := d.awsECS.DescribeTaskDefinition(params)
	if err != nil {
		return nil, err
	}

	return resp.TaskDefinition, nil
}

// RegisterTaskDefinition register new task definition if needed
func (d *Deploy) RegisterTaskDefinition(baseDefinition *ecs.TaskDefinition) (*ecs.TaskDefinition, error) {
	var containerDefinitions []*ecs.ContainerDefinition
	for _, c := range baseDefinition.ContainerDefinitions {
		newDefinition, err := d.NewContainerDefinition(c)
		if err != nil {
			return nil, err
		}
		containerDefinitions = append(containerDefinitions, newDefinition)
	}
	params := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: containerDefinitions,
		Family:               baseDefinition.Family,
		NetworkMode:          baseDefinition.NetworkMode,
		PlacementConstraints: baseDefinition.PlacementConstraints,
		TaskRoleArn:          baseDefinition.TaskRoleArn,
		Volumes:              baseDefinition.Volumes,
	}

	resp, err := d.awsECS.RegisterTaskDefinition(params)
	if err != nil {
		return nil, err
	}

	return resp.TaskDefinition, nil
}

// NewContainerDefinition update image tag in a given container definition.
// If the container definition is not target container, return givien definition.
func (d *Deploy) NewContainerDefinition(baseDefinition *ecs.ContainerDefinition) (*ecs.ContainerDefinition, error) {
	if d.newTask.image == nil {
		return baseDefinition, nil
	}
	baseRepository, _, err := divideImageAndTag(*baseDefinition.Image)
	if err != nil {
		return nil, err
	}
	if d.newTask.image.repository != *baseRepository {
		return baseDefinition, nil
	}
	imageWithTag := (d.newTask.image.repository) + ":" + (d.newTask.image.tag)
	baseDefinition.Image = &imageWithTag
	return baseDefinition, nil
}