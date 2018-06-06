package main

import (
	"context"
	"sort"
	"testing"
	"time"

	codeship "github.com/codeship/codeship-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type mockBuildGetter struct {
	buildStatus string
}

func (m mockBuildGetter) ListBuilds(ctx context.Context, projectUUID string, opts ...codeship.PaginationOption) (codeship.BuildList, codeship.Response, error) {
	now := time.Now()
	build1 := codeship.Build{UUID: "2", Status: "testing", Branch: "test-branch", AllocatedAt: now.Add(time.Duration(-1) * time.Minute)}
	build2 := codeship.Build{UUID: "3", Status: "success", Branch: "test-branch", AllocatedAt: now}
	build3 := codeship.Build{UUID: "1", Status: "testing", Branch: "test-branch", AllocatedAt: now.Add(time.Duration(-5) * time.Minute)}
	build4 := codeship.Build{UUID: "4", Status: "testing", Branch: "another-branch"}

	return codeship.BuildList{
		Builds: []codeship.Build{build1, build2, build3, build4},
	}, codeship.Response{}, nil
}

func (m mockBuildGetter) GetBuild(ctx context.Context, projectUUID, buildUUID string) (codeship.Build, codeship.Response, error) {
	return codeship.Build{
		Branch: "test-branch",
		Status: m.buildStatus,
	}, codeship.Response{}, nil
}

func TestWaitOnPreviousBuilds(t *testing.T) {
	monitor := &monitor{
		buildGetter: mockBuildGetter{},
	}

	err := monitor.waitOnPreviousBuilds(context.TODO(), "project-uuid", "build-uuid", "test-branch")
	require.NoError(t, err)
}

func TestBuildFinished(t *testing.T) {
	testCases := []struct {
		name        string
		buildStatus string
		finished    bool
	}{
		{
			name:        "success status",
			buildStatus: "success",
			finished:    true,
		}, {
			name:        "testing status",
			buildStatus: "testing",
			finished:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := codeship.Build{
				ProjectUUID: "project-uuid",
				UUID:        "build-uuid",
			}

			monitor := &monitor{
				buildGetter: mockBuildGetter{
					buildStatus: tc.buildStatus,
				},
			}

			finished, err := monitor.buildFinished(context.TODO(), b)
			require.NoError(t, err)
			assert.Equal(t, finished, tc.finished)
		})
	}
}

func TestBuildsToWatch(t *testing.T) {
	monitor := &monitor{
		buildGetter: mockBuildGetter{},
	}

	builds, err := monitor.buildsToWatch(context.TODO(), "project-id", "test-branch")
	require.NoError(t, err)
	assert.Equal(t, len(builds), 2)
}
