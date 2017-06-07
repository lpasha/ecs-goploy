package cmd

import (
	"log"
	"time"

	ecsdeploy "github.com/crowdworks/ecs-goploy/deploy"
	"github.com/spf13/cobra"
)

type deploy struct {
	cluster        string
	name           string
	imageWithTag   string
	profile        string
	region         string
	timeout        int
	enableRollback bool
}

func deployCmd() *cobra.Command {
	d := &deploy{}
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy ECS",
		Run:   d.deploy,
	}

	flags := cmd.Flags()
	flags.StringVarP(&d.cluster, "cluster", "c", "", "Name of ECS cluster")
	flags.StringVarP(&d.name, "service-name", "n", "", "Name of service to deploy")
	flags.StringVarP(&d.imageWithTag, "image", "i", "", "Name of Docker image to run, ex: repo/image:latest")
	flags.StringVarP(&d.profile, "profile", "p", "", "AWS Profile to use")
	flags.StringVarP(&d.region, "region", "r", "", "AWS Region Name")
	flags.IntVarP(&d.timeout, "timeout", "t", 300, "Timeout seconds. Script monitors ECS Service for new task definition to be running")
	flags.BoolVar(&d.enableRollback, "enable-rollback", false, "Rollback task definition if new version is not running before TIMEOUT")

	return cmd
}

func (d *deploy) deploy(cmd *cobra.Command, args []string) {
	e := ecsdeploy.NewDeploy(d.cluster, d.name, d.profile, d.region, d.imageWithTag, (time.Duration(d.timeout) * time.Second), d.enableRollback)
	if err := e.Deploy(); err != nil {
		log.Fatalf("[ERROR] %v", err)
	}
	log.Println("[INFO] Deploy success")
}