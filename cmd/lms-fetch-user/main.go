package main

import (
    "fmt"

    "github.com/alecthomas/kong"
    "github.com/rs/zerolog/log"

    "github.com/eriq-augustine/autograder/config"
    "github.com/eriq-augustine/autograder/db"
    "github.com/eriq-augustine/autograder/lms"
)

var args struct {
    config.ConfigArgs
    Course string `help:"ID of the course." arg:""`
    Email string `help:"Email of the user to fetch." arg:""`
}

func main() {
    kong.Parse(&args,
        kong.Description("Fetch users for a specific LMS course."),
    );

    err := config.HandleConfigArgs(args.ConfigArgs);
    if (err != nil) {
        log.Fatal().Err(err).Msg("Could not load config options.");
    }

    db.MustOpen();
    defer db.MustClose();

    course := db.MustGetCourse(args.Course);

    user, err := lms.FetchUser(course, args.Email);
    if (err != nil) {
        log.Fatal().Err(err).Msg("Could not fetch user.");
    }

    if (user == nil) {
        fmt.Println("No user found.");
        return;
    }

    fmt.Println("id\temail\tname\trole");
    fmt.Printf("%s\t%s\t%s\t%s\n", user.ID, user.Email, user.Name, user.Role.String());
}
