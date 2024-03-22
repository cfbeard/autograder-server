package grader

import (
    "github.com/edulinq/autograder/db"
    "github.com/edulinq/autograder/docker"
    "github.com/edulinq/autograder/util"
    "testing"
)

func TestLogConfig(test *testing.T) {
    db.ResetForTesting()
    defer db.ResetForTesting()

    assignment := db.MustGetTestAssignment()

    tempDir, err := util.MkDirTemp("autograder-test-docker-log-");
    if err != nil {
        test.Fatal(err)
    }
    defer util.RemoveDirent(tempDir)

    err = docker.BuildImage(assignment)
    if err != nil {
        test.Fatal(err)
    }

    stdout, stderr, err := docker.RunContainer(assignment, assignment.ImageName(), tempDir, tempDir, "test")
    if err != nil {
        test.Fatal(err)
    }

    test.Log(stdout, stderr)

}
