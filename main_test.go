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
