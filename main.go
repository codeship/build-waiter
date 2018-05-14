package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"time"

	codeship "github.com/codeship/codeship-go"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type allocatedAtSort []codeship.Build

func (aas allocatedAtSort) Len() int {
	return len(aas)
}
func (aas allocatedAtSort) Swap(i, j int) {
	aas[i], aas[j] = aas[j], aas[i]
}
func (aas allocatedAtSort) Less(i, j int) bool {
	return aas[i].AllocatedAt.Before(aas[j].AllocatedAt)
}

func main() {
	log.SetFlags(0)

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		log.Fatal(err)
	}

	viper.SetEnvPrefix("codeship")

	// CODESHIP_USERNAME
	err = viper.BindEnv("username")
	if err != nil {
		log.Fatal(err)
	}

	// CODESHIP_PASSWORD
	err = viper.BindEnv("password")
	if err != nil {
		log.Fatal(err)
	}

	// CODESHIP_ORGANIZATION
	err = viper.BindEnv("organization")
	if err != nil {
		log.Fatal(err)
	}

	//CI_PROJECT_ID
	err = viper.BindEnv("project_id", "CI_PROJECT_ID")
	if err != nil {
		log.Fatal(err)
	}

	err = viper.BindEnv("build_id", "CI_BUILD_ID")
	if err != nil {
		log.Fatal(err)
	}

	user := viper.GetString("username")
	if user == "" {
		log.Fatal("CODESHIP_USERNAME required")
	}

	password := viper.GetString("password")
	if password == "" {
		log.Fatal("CODESHIP_PASSWORD required")
	}

	orgName := viper.GetString("organization")
	if orgName == "" {
		log.Fatal("CODESHIP_ORGANIZATION required")
	}

	projectUUID := viper.GetString("project_id")
	if projectUUID == "" {
		log.Fatal("CI_PROJECT_ID required")
	}

	buildUUID := viper.GetString("build_id")
	if buildUUID == "" {
		log.Fatal("CI_BUILD_ID required")
	}

	ctx := context.Background()
	// trap Ctrl+C and call cancel on the context
	ctx, cancel := context.WithCancel(ctx)
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		cancel()
	}()

	auth := codeship.NewBasicAuth(user, password)
	client, err := codeship.New(auth)
	if err != nil {
		log.Fatal(err)
	}

	org, err := client.Organization(ctx, "codeship")
	if err != nil {
		log.Fatal(err)
	}

	wb, err := buildsToWatch(ctx, org, projectUUID)
	if err != nil {
		log.Fatal(err)
	}

	sort.Sort(allocatedAtSort(wb))

	for _, b := range wb {
		fmt.Println("UUID>>", b.UUID)
		if b.UUID == buildUUID {
			fmt.Println("my turn....exiting")
			// It is our turn to run --exit
			break
		} else {
			// wait the build ahead of us to finish
			for {
				finished, err := buildFinished(ctx, org, b)
				if err != nil {
					log.Fatal(err)
				}
				if finished {
					break
				} else {
					fmt.Println("Waiting on build", b.UUID)
					time.Sleep(30 * time.Second)
				}
			}
		}
	}
}

func buildFinished(ctx context.Context, org *codeship.Organization, b codeship.Build) (bool, error) {
	nb, _, err := org.GetBuild(ctx, b.ProjectUUID, b.UUID)
	if err != nil {
		return false, err
	}

	return (nb.Status != "testing"), nil
}

func buildsToWatch(ctx context.Context, org *codeship.Organization, projectUUID string) ([]codeship.Build, error) {
	wb := []codeship.Build{}

	build_list, resp, err := org.ListBuilds(ctx, projectUUID)
	if err != nil {
		return nil, err
	}

	pageWithRunningBuild := true
	for {
		if resp.IsLastPage() || resp.Next == "" {
			break
		}

		pageWithRunningBuild = false
		for _, b := range build_list.Builds {
			if b.Status == "testing" {
				pageWithRunningBuild = true

				if b.Branch == "sleep" {
					wb = append(wb, b)
				}
			}
		}

		if !pageWithRunningBuild {
			break
		}

		next, _ := resp.NextPage()

		build_list, resp, err = org.ListBuilds(ctx, projectUUID, codeship.Page(next), codeship.PerPage(50))
		if err != nil {
			return nil, err
		}
		break
	}

	return wb, nil
}
