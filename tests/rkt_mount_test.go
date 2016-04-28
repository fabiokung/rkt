// Copyright 2016 The rkt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build coreos src

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/coreos/rkt/tests/testutils"
)

func TestMountSymlink(t *testing.T) {
	tmpDir := createTempDirOrPanic("rkt-mount-test-")
	defer os.RemoveAll(tmpDir)
	mountSrcFile := filepath.Join(tmpDir, "hello")
	if err := ioutil.WriteFile(mountSrcFile, []byte("world"), 0666); err != nil {
		t.Fatalf("Cannot write file: %v", err)
	}

	image := patchTestACI("rkt-test-mount-symlink.aci", "--exec=/inspect --read-file")
	defer os.Remove(image)

	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	tests := []struct {
		linkFile     string
		actualFile   string
		expectedLine string
		exitCode     int
	}{
		// '/dir1/link_abs_dir2' links to '/dir2'.
		{
			"/dir1/link_abs_dir2/foo",
			"/dir2/foo",
			"world",
			0,
		},
		// '/dir1/link_rel_dir2' links to '/dir2'.
		{
			"/dir1/link_rel_dir2/bar",
			"/dir2/bar",
			"world",
			0,
		},
		// '/dir1/../dir1/.//link_rel_dir2' links to '/dir2'.
		{
			"/dir1/../dir1/.//link_rel_dir2/foo",
			"/dir2/foo",
			"world",
			0,
		},
		// '/dir1/../../../foo' is invalid because it tries to escape rootfs.
		{
			"/dir1/../../../foo",
			"/dir2/foo",
			"path escapes app's root",
			1,
		},
		// '/dir1/link_invalid' is an invalid link because it tries to escape rootfs.
		{
			"/dir1/link_invalid/foo",
			"/dir2/foo",
			"escapes app's root with value",
			1,
		},
	}

	for _, tt := range tests {
		paramMount := fmt.Sprintf("--volume=test,kind=host,source=%s --mount volume=test,target=%s", mountSrcFile, tt.linkFile)

		// Read the actual file.
		rktCmd := fmt.Sprintf("%s --insecure-options=image run %s --set-env=FILE=%s %s", ctx.Cmd(), paramMount, tt.actualFile, image)
		t.Logf("%s\n", rktCmd)

		if tt.exitCode == 0 {
			runRktAndCheckOutput(t, rktCmd, tt.expectedLine, false)
		} else {
			child := spawnOrFail(t, rktCmd)
			result, out, err := expectRegexWithOutput(child, tt.expectedLine)
			if err != nil || len(result) != 1 {
				t.Errorf("%q regex must be found one time, Error: %v\nOutput: %v", tt.expectedLine, err, out)
			}
			waitOrFail(t, child, tt.exitCode)
		}
	}
}
