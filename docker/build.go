package docker

// Handle building docker images for grading.

import (
    "bufio"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"

	"github.com/docker/docker/api/types"
    "github.com/docker/docker/pkg/archive"
    "github.com/rs/zerolog/log"

    "github.com/eriq-augustine/autograder/common"
    "github.com/eriq-augustine/autograder/config"
    "github.com/eriq-augustine/autograder/util"
)

const (
    TEMPDIR_PREFIX = "autograder-docker-build-";
)

type BuildOptions struct {
    Rebuild bool `help:"Rebuild images ignoring caches." default:"false"`
}

func NewBuildOptions() *BuildOptions {
    return &BuildOptions{
        Rebuild: false,
    };
}

func BuildImage(imageInfo *ImageInfo) error {
    return BuildImageWithOptions(imageInfo, NewBuildOptions());
}

func BuildImageWithOptions(imageInfo *ImageInfo, options *BuildOptions) error {
    tempDir, err := os.MkdirTemp("", TEMPDIR_PREFIX + imageInfo.Name + "-");
    if (err != nil) {
        return fmt.Errorf("Failed to create temp build directory for '%s': '%w'.", imageInfo.Name, err);
    }

    if (config.DEBUG.GetBool()) {
        log.Info().Str("path", tempDir).Msg("Leaving behind temp building dir.");
    } else {
        defer os.RemoveAll(tempDir);
    }

    err = writeDockerContext(imageInfo, tempDir);
    if (err != nil) {
        return err;
    }

    buildOptions := types.ImageBuildOptions{
        Tags: []string{imageInfo.Name},
        Dockerfile: "Dockerfile",
    };

    if (options.Rebuild) {
        buildOptions.NoCache = true;
    }

    // Create the build context by adding all the relevant files.
    tar, err := archive.TarWithOptions(tempDir, &archive.TarOptions{});
    if (err != nil) {
        return fmt.Errorf("Failed to create tar build context for image '%s': '%w'.", imageInfo.Name, err);
    }

    return buildImage(buildOptions, tar);
}

func buildImage(buildOptions types.ImageBuildOptions, tar io.ReadCloser) error {
	ctx, docker, err := getDockerClient();
    if (err != nil) {
        return err;
    }
	defer docker.Close()

    response, err := docker.ImageBuild(ctx, tar, buildOptions);
    if (err != nil) {
        return fmt.Errorf("Failed to run docker image build command: '%w'.", err);
    }

    output := collectBuildOutput(response);
    log.Debug().Str("image-build-output", output).Msg("Image Build Output");

    return nil;
}

// Try to get the build output from a build response.
// Note that the response may be from a failure.
func collectBuildOutput(response types.ImageBuildResponse) string {
    if (response.Body == nil) {
        return "";
    }

    defer response.Body.Close();

    buildStringOutput := strings.Builder{};

    responseScanner := bufio.NewScanner(response.Body);
    for responseScanner.Scan() {
        line := responseScanner.Text();

        line = strings.TrimSpace(line);
        if (line == "") {
            continue;
        }

        jsonData, err := util.JSONMapFromString(line);
        if (err != nil) {
            buildStringOutput.WriteString("<WARNING: The following output line was not JSON.>");
            buildStringOutput.WriteString(line);
        }

        rawText, ok := jsonData["error"];
        if (ok) {
            text, ok := rawText.(string);
            if (!ok) {
                text = "<ERROR: Docker output JSON value is not a string.>";
            }

            log.Warn().Err(err).Str("message", text).Msg("Docker image build had an error entry.");
            buildStringOutput.WriteString(text);
        }

        rawText, ok = jsonData["stream"];
        if (ok) {
            text, ok := rawText.(string);
            if (!ok) {
                text = "<ERROR: Docker output JSON value is not a string.>";
            }

            buildStringOutput.WriteString(text);
        }
    }

    err := responseScanner.Err();
    if (err != nil) {
        log.Warn().Err(err).Msg("Failed to scan docker image build response.");
    }

    return buildStringOutput.String();
}

// Write a full docker build context (Dockerfile and static files) to the given directory.
func writeDockerContext(imageInfo *ImageInfo, dir string) error {
    _, _, workDir, err := common.CreateStandardGradingDirs(dir);
    if (err != nil) {
        return fmt.Errorf("Could not create standard grading directories: '%w'.", err);
    }

    // Copy over the static files (and do any file ops).
    err = common.CopyFileSpecs(imageInfo.BaseDir, workDir, dir,
            imageInfo.StaticFiles, false, imageInfo.PreStaticFileOperations, imageInfo.PostStaticFileOperations);
    if (err != nil) {
        return fmt.Errorf("Failed to copy static imageInfo files: '%w'.", err);
    }

    dockerConfigPath := filepath.Join(dir, DOCKER_CONFIG_FILENAME);
    err = util.ToJSONFile(imageInfo.GetGradingConfig(), dockerConfigPath);
    if (err != nil) {
        return fmt.Errorf("Failed to create docker config file: '%w'.", err);
    }

    dockerfilePath := filepath.Join(dir, "Dockerfile");
    err = writeDockerfile(imageInfo, workDir, dockerfilePath)
    if (err != nil) {
        return err;
    }

    return nil;
}

func writeDockerfile(imageInfo *ImageInfo, workDir string, path string) error {
    contents, err := toDockerfile(imageInfo, workDir)
    if (err != nil) {
        return fmt.Errorf("Failed get contenets for dockerfile ('%s'): '%w'.", path, err);
    }

    err = util.WriteFile(contents, path);
    if (err != nil) {
        return fmt.Errorf("Failed write dockerfile ('%s'): '%w'.", path, err);
    }

    return nil;
}

func toDockerfile(imageInfo *ImageInfo, workDir string) (string, error) {
    // Note that we will insert blank lines for formatting.
    lines := make([]string, 0);

    lines = append(lines, fmt.Sprintf("FROM %s", imageInfo.Image), "")

    // Ensure standard directories are created.
    lines = append(lines, "# Core directories");
    for _, dir := range []string{DOCKER_BASE_DIR, DOCKER_INPUT_DIR, DOCKER_OUTPUT_DIR, DOCKER_WORK_DIR} {
        lines = append(lines, fmt.Sprintf("RUN mkdir -p '%s'", dir));
    }
    lines = append(lines, "");

    // Set the working directory.
    lines = append(lines, fmt.Sprintf("WORKDIR %s", DOCKER_BASE_DIR), "")

    // Copy over the config file.
    lines = append(lines, fmt.Sprintf("COPY %s %s", DOCKER_CONFIG_FILENAME, DOCKER_CONFIG_PATH), "");

    // Append pre-static docker commands.
    lines = append(lines, "# Pre-Static Commands");
    lines = append(lines, imageInfo.PreStaticDockerCommands...);
    lines = append(lines, "");

    // Copy over all the contents of the work directory (this is after post-static file ops).
    dirents, err := os.ReadDir(workDir);
    if (err != nil) {
        return "", fmt.Errorf("Failed to list work dir ('%s') for static files: '%w'.", workDir, err);
    }

    lines = append(lines, "# Static Files");
    for _, dirent := range dirents {
        sourcePath := DockerfilePathQuote(filepath.Join(common.GRADING_WORK_DIRNAME, dirent.Name()));
        destPath := DockerfilePathQuote(filepath.Join(DOCKER_WORK_DIR, dirent.Name()));

        lines = append(lines, fmt.Sprintf("COPY %s %s", sourcePath, destPath));
    }
    lines = append(lines, "");

    // Append post-static docker commands.
    lines = append(lines, "# Post-Static Commands");
    lines = append(lines, imageInfo.PostStaticDockerCommands...);
    lines = append(lines, "");

    return strings.Join(lines, "\n"), nil;
}