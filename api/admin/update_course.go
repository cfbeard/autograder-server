package admin

import (
    "github.com/eriq-augustine/autograder/api/core"
    "github.com/eriq-augustine/autograder/db"
)

type UpdateCourseRequest struct {
    core.APIRequestCourseUserContext
    core.MinRoleAdmin

    Clear bool `json:"clear"`
}

type UpdateCourseResponse struct {
    CourseUpdated bool `json:"course-updated"`
}

func HandleUpdateCourse(request *UpdateCourseRequest) (*UpdateCourseResponse, *core.APIError) {
    if (request.Clear) {
        err := db.ClearCourse(request.Course);
        if (err != nil) {
            return nil, core.NewInternalError("-701", &request.APIRequestCourseUserContext,
                    "Failed to clear course.").Err(err);
        }
    }

    _, updated, err := db.UpdateCourseFromSource(request.Course);
    if (err != nil) {
        return nil, core.NewInternalError("-702", &request.APIRequestCourseUserContext,
                "Failed to reload course.").Err(err);
    }

    return &UpdateCourseResponse{updated}, nil;
}