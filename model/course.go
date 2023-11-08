package model

import (
    "github.com/eriq-augustine/autograder/docker"
    // TEST
    // "github.com/eriq-augustine/autograder/usr"
)

type Course interface {
    GetID() string
    GetName() string
    GetSourceDir() string
    GetLMSAdapter() *LMSAdapter
    HasAssignment(id string) bool;
    GetAssignment(id string) Assignment;
    GetAssignments() map[string]Assignment;
    GetSortedAssignments() []Assignment
    GetAssignmentLMSIDs() ([]string, []string)

    /* TEST
    GetUser(email string) (*usr.User, error);
    GetUsers() (map[string]*usr.User, error)
    // TODO(eriq): Save a single user.
    SaveUsers(users map[string]*usr.User) error;
    AddUser(user *usr.User, merge bool, dryRun bool, sendEmails bool) (*usr.UserSyncResult, error);
    SyncNewUsers(newUsers map[string]*usr.User, merge bool, dryRun bool, sendEmails bool) (*usr.UserSyncResult, error);
    */

    Activate() error;
    BuildAssignmentImages(force bool, quick bool, options *docker.BuildOptions) ([]string, map[string]error);
    GetCacheDir() string;

    SetSourcePathForTesting(sourcePath string) string;
}
