package main

import (
	"context"
	"sort"
	"testing"
	"time"

	codeship "github.com/codeship/codeship-go"
	"github.com/stretchr/testify/assert"
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

func TestWaitOnPreviousBuilds(t *testing.T) {
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

type testBuildGetter struct {
	testBuildStatus string
}

func (bg testBuildGetter) ListBuilds(ctx context.Context, projectUUID string, opts ...codeship.PaginationOption) (codeship.BuildList, codeship.Response, error) {
	build1 := codeship.Build{Status: "testing", Branch: "test-branch"}
	build2 := codeship.Build{Status: "success", Branch: "test-branch"}
	build3 := codeship.Build{Status: "testing", Branch: "test-branch"}
	build4 := codeship.Build{Status: "testing", Branch: "another-branch"}

	bl := codeship.BuildList{
		Builds: []codeship.Build{build1, build2, build3, build4},
	}
	r := codeship.Response{}

	return bl, r, nil
}

func (bg testBuildGetter) GetBuild(ctx context.Context, projectUUID, buildUUID string) (codeship.Build, codeship.Response, error) {
	b := codeship.Build{
		Branch: "test-branch",
		Status: bg.testBuildStatus,
	}

	r := codeship.Response{}

	return b, r, nil
}

func TestBranchBuild(t *testing.T) {
	ctx := context.Background()
	projectUUID := "my-project-uuid"
	buildUUID := "my-build-uuid"

	tbg := testBuildGetter{}
	branch, err := buildBranch(ctx, tbg, projectUUID, buildUUID)
	assert.Nil(t, err)
	assert.Equal(t, branch, "test-branch")
}

func TestBuildFinishedWithSuccessStatus(t *testing.T) {
	ctx := context.Background()
	projectUUID := "my-project-uuid"
	buildUUID := "my-build-uuid"

	b := codeship.Build{
		ProjectUUID: projectUUID,
		UUID:        buildUUID,
	}

	bm := codeshipBuildMonitor{
		bg: testBuildGetter{testBuildStatus: "success"},
	}

	finished, err := bm.buildFinished(ctx, b)
	assert.Nil(t, err)
	assert.Equal(t, finished, true)
}

func TestBuildFinishedWithTestingStatus(t *testing.T) {
	ctx := context.Background()
	projectUUID := "my-project-uuid"
	buildUUID := "my-build-uuid"

	b := codeship.Build{
		ProjectUUID: projectUUID,
		UUID:        buildUUID,
	}

	bm := codeshipBuildMonitor{
		bg: testBuildGetter{testBuildStatus: "testing"},
	}

	finished, err := bm.buildFinished(ctx, b)
	assert.Nil(t, err)
	assert.Equal(t, finished, false)
}

func TestBuildsToWatch(t *testing.T) {
	ctx := context.Background()
	projectUUID := "my-project-uuid"

	tbg := testBuildGetter{}

	bm := codeshipBuildMonitor{
		bg: tbg,
	}

	l, err := bm.buildsToWatch(ctx, projectUUID, "test-branch")
	assert.Nil(t, err)
	assert.Equal(t, len(l), 2)

}
