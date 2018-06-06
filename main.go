package main

import (
	"context"
	"flag"
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

func (s allocatedAtSort) Len() int {
	return len(s)
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

	// CI_PROJECT_ID
	err = viper.BindEnv("project_id", "CI_PROJECT_ID")
	if err != nil {
		log.Fatal(err)
	}

	// CI_BUILD_ID
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
	defer func() {
		signal.Stop(c)
		close(c)
		cancel()
	}()

	go func() {
		<-c
		cancel()
	}()

	auth := codeship.NewBasicAuth(user, password)
	client, err := codeship.New(auth)
	if err != nil {
		log.Fatal(err)
	}

	org, err := client.Organization(ctx, orgName)
	if err != nil {
		log.Fatal(err)
	}

	build, _, err := org.GetBuild(ctx, projectUUID, buildUUID)
	if err != nil {
		log.Fatal(err)
	}

	bm := monitor{
		buildGetter: org,
	}

	err = bm.waitOnPreviousBuilds(ctx, projectUUID, buildUUID, build.Branch)
	if err != nil {
		log.Fatal(err)
	}
}

type buildGetter interface {
	ListBuilds(ctx context.Context, projectUUID string, opts ...codeship.PaginationOption) (codeship.BuildList, codeship.Response, error)
	GetBuild(context.Context, string, string) (codeship.Build, codeship.Response, error)
}

type monitor struct {
	buildGetter
}

func (m monitor) waitOnPreviousBuilds(ctx context.Context, projectUUID, buildUUID, branch string) error {
	// Find a list all builds running for the branch
	watching, err := m.buildsToWatch(ctx, projectUUID, branch)
	if err != nil {
		return err
	}

	// Sort builds by oldest allocated time
	sort.Sort(allocatedAtSort(watching))

	// Loop through list of builds on branch.
	// Check every 30 seconds to see if build has completed
	// exit out of loop when we reach out build
	ticker := time.NewTicker(30 * time.Second)
	for _, b := range watching {
		if b.UUID == buildUUID {
			// It is our turn to run --exit
			log.Println("Resuming build")
			break
		} else {
			// wait for the build ahead of us to finish
			finished, err := m.buildFinished(ctx, b)
			if err != nil {
				return err
			}
			if finished {
				continue
			} else {
				log.Println("Waiting on build", b.UUID)
			}
		BuildWait:
			for {
				select {
				case <-ctx.Done():
					return nil // user has hit ctrl+c
				case <-ticker.C:
					finished, err := m.buildFinished(ctx, b)
					if err != nil {
						return err
					}
					if finished {
						break BuildWait
					} else {
						log.Println("Waiting on build", b.UUID)
					}
				}
			}
		}
	}
	return nil
}

func (m monitor) buildFinished(ctx context.Context, b codeship.Build) (bool, error) {
	build, _, err := m.GetBuild(ctx, b.ProjectUUID, b.UUID)
	if err != nil {
		return false, err
	}

	// a build is considered finished if it is not testing
	return (build.Status != "testing"), nil
}

func (m monitor) buildsToWatch(ctx context.Context, projectUUID, branch string) ([]codeship.Build, error) {
	var (
		pageWithRunningBuild bool
		watching             []codeship.Build
	)

	builds, resp, err := m.ListBuilds(ctx, projectUUID)
	if err != nil {
		return nil, err
	}

	// loop through builds until we get to a page without any running builds or we reach the last page
	for {
		pageWithRunningBuild = false
		for _, b := range builds.Builds {
			if b.Status == "testing" {
				pageWithRunningBuild = true
				if b.Branch == branch {
					watching = append(watching, b)
				}
			}
		}

		if resp.IsLastPage() || resp.Next == "" {
			break
		}

		if !pageWithRunningBuild {
			break
		}

		next, _ := resp.NextPage()

		builds, resp, err = m.ListBuilds(ctx, projectUUID, codeship.Page(next), codeship.PerPage(50))
		if err != nil {
			return nil, err
		}
	}

	return watching, nil
}
