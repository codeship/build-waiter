package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	codeship "github.com/codeship/codeship-go"
	"github.com/stretchr/testify/assert"
)

var (
	mux    *http.ServeMux
	server *httptest.Server
	client *codeship.Client
	org    *codeship.Organization
)

func TestAllocatedAtSort(t *testing.T) {
	now := time.Now()
	build1 := codeship.Build{AllocatedAt: now.Add(time.Duration(-1) * time.Minute)}
	build2 := codeship.Build{AllocatedAt: now}
	build3 := codeship.Build{AllocatedAt: now.Add(time.Duration(-5) * time.Minute)}

	bl := []codeship.Build{build1, build2, build3}

	sort.Sort(allocatedAtSort(bl))

	assert.Equal(t, bl[0], build3)
	assert.Equal(t, bl[1], build1)
	assert.Equal(t, bl[2], build2)
}

type fakeBuildMonitor struct {
	finishedBuildCalls      *[]codeship.Build
	buildToWatchProjectUUID string
	buildToWatchBranch      string
}

func (bm *fakeBuildMonitor) buildFinished(ctx context.Context, b codeship.Build) (bool, error) {
	finishedCall := append(*bm.finishedBuildCalls, b)
	bm.finishedBuildCalls = &finishedCall

	return true, nil
}

func (bm *fakeBuildMonitor) buildsToWatch(ctx context.Context, projectUUID, branch string) ([]codeship.Build, error) {

	bm.buildToWatchProjectUUID = projectUUID
	bm.buildToWatchBranch = branch

	now := time.Now()
	build1 := codeship.Build{UUID: "2", AllocatedAt: now.Add(time.Duration(-1) * time.Minute)}
	build2 := codeship.Build{UUID: "3", AllocatedAt: now}
	build3 := codeship.Build{UUID: "1", AllocatedAt: now.Add(time.Duration(-5) * time.Minute)}

	return []codeship.Build{build3, build1, build2}, nil
}

func TestWaitOnPreviousbuilds(t *testing.T) {
	ctx := context.Background()
	builds := []codeship.Build{}
	bm := &fakeBuildMonitor{finishedBuildCalls: &builds}

	waitOnPreviousBuilds(ctx, bm, "project-uuid", "build-uuid", "my-branch")

	assert.Equal(t, bm.buildToWatchProjectUUID, "project-uuid")
	assert.Equal(t, bm.buildToWatchBranch, "my-branch")

	finishedBuilds := *bm.finishedBuildCalls
	assert.Equal(t, finishedBuilds[0].UUID, "1")
	assert.Equal(t, finishedBuilds[1].UUID, "2")
	assert.Equal(t, finishedBuilds[2].UUID, "3")
}

func TestBranchBuild(t *testing.T) {
	setup()
	ctx := context.Background()
	projectUUID := "my-project-uuid"
	buildUUID := "my-build-uuid"

	mux.HandleFunc(fmt.Sprintf("/organizations/%s/projects/%s/builds/%s",
		org.UUID,
		projectUUID,
		buildUUID),
		func(w http.ResponseWriter, r *http.Request) {
			assert := assert.New(t)
			assert.Equal("GET", r.Method)
			assert.Equal("application/json", r.Header.Get("Content-Type"))
			assert.Equal("application/json", r.Header.Get("Accept"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, fixture("builds/get.json"))

		})

	branch, err := buildBranch(ctx, org, projectUUID, buildUUID)
	assert.Nil(t, err)
	assert.Equal(t, branch, "test-branch")
}

func TestBuildFinishedWithSuccessStatus(t *testing.T) {
	setup()
	ctx := context.Background()
	projectUUID := "my-project-uuid"
	buildUUID := "my-build-uuid"

	mux.HandleFunc(fmt.Sprintf("/organizations/%s/projects/%s/builds/%s",
		org.UUID,
		projectUUID,
		buildUUID),
		func(w http.ResponseWriter, r *http.Request) {
			assert := assert.New(t)
			assert.Equal("GET", r.Method)
			assert.Equal("application/json", r.Header.Get("Content-Type"))
			assert.Equal("application/json", r.Header.Get("Accept"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, fixture("builds/get.json"))

		})

	b := codeship.Build{
		ProjectUUID: projectUUID,
		UUID:        buildUUID,
	}

	bm := CodeshipBuildMonitor{
		org: org,
	}

	finished, err := bm.buildFinished(ctx, b)
	assert.Nil(t, err)
	assert.Equal(t, finished, true)
}

func TestBuildFinishedWithTestingStatus(t *testing.T) {
	setup()
	ctx := context.Background()
	projectUUID := "my-project-uuid"
	buildUUID := "my-build-uuid"

	mux.HandleFunc(fmt.Sprintf("/organizations/%s/projects/%s/builds/%s",
		org.UUID,
		projectUUID,
		buildUUID),
		func(w http.ResponseWriter, r *http.Request) {
			assert := assert.New(t)
			assert.Equal("GET", r.Method)
			assert.Equal("application/json", r.Header.Get("Content-Type"))
			assert.Equal("application/json", r.Header.Get("Accept"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, fixture("builds/get-testing.json"))

		})

	b := codeship.Build{
		ProjectUUID: projectUUID,
		UUID:        buildUUID,
	}

	bm := CodeshipBuildMonitor{
		org: org,
	}

	finished, err := bm.buildFinished(ctx, b)
	assert.Nil(t, err)
	assert.Equal(t, finished, false)
}

func TestBuildsToWatch(t *testing.T) {
	setup()
	ctx := context.Background()
	projectUUID := "my-project-uuid"

	mux.HandleFunc(fmt.Sprintf("/organizations/%s/projects/%s/builds",
		org.UUID,
		projectUUID),
		func(w http.ResponseWriter, r *http.Request) {
			assert := assert.New(t)
			assert.Equal("GET", r.Method)
			assert.Equal("application/json", r.Header.Get("Content-Type"))
			assert.Equal("application/json", r.Header.Get("Accept"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, fixture("builds/list.json"))

		})

	bm := CodeshipBuildMonitor{
		org: org,
	}

	l, err := bm.buildsToWatch(ctx, projectUUID, "test-branch")
	assert.Nil(t, err)
	assert.Equal(t, len(l), 2)

}

func setup() func() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	mux.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, fixture("auth/success.json"))
	})

	client, _ = codeship.New(codeship.NewBasicAuth("test", "pass"), codeship.BaseURL(server.URL))
	org, _ = client.Organization(context.Background(), "codeship")

	return func() {
		server.Close()
	}
}

func fixture(path string) string {
	b, err := ioutil.ReadFile("testdata/fixtures/" + path)
	if err != nil {
		panic(err)
	}
	return string(b)
}
