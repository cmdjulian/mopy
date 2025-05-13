package llb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/containerd/containerd/platforms"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/moby/buildkit/frontend/dockerfile/dockerfile2llb"
	"github.com/moby/buildkit/frontend/dockerui" // For dockerui.Config and dockerui.Client
	gatewayclient "github.com/moby/buildkit/frontend/gateway/client"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"gitlab.com/cmdjulian/mopy/pkg/config" // Your project's config package
	"golang.org/x/sync/errgroup"
)

const (
	defaultDockerfileName = "Mopyfile.yaml"
	// It's assumed Mopyfile.yaml resides within this main build context.
	localNameContext  = "context"
	keyCacheFrom      = "cache-from"
	keyCacheImports   = "cache-imports"
	keyConfigPath     = "filename" // User-provided option for the Mopyfile path within the context
	keyTargetPlatform = "platform"
)

// Build is the main function for your custom BuildKit frontend.
// It reads a Mopyfile, converts it to Dockerfile content, then to LLB, and solves it.
func Build(ctx context.Context, c gatewayclient.Client) (*gatewayclient.Result, error) {
	// 1. Load your Mopyfile configuration.
	// Assumes Mopyfile.yaml (or path from keyConfigPath) is in the main build context.
	mopyConfig, err := readMopyConfig(ctx, c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load mopy configuration")
	}

	// 2. Convert Mopyfile config to Dockerfile content string using your custom logic.
	dockerfileContent := Mopyfile2LLB(mopyConfig)

	buildOpts := c.BuildOpts()
	opts := buildOpts.Opts // Raw build options (like --build-arg, --platform) from the client.

	// 3. Initialize dockerui.Client. This is crucial for standard frontend behaviors.
	// - It parses global build opts (like --build-arg) into duc.Config.
	// - It provides duc.MainContext(), which loads the primary build context and handles .dockerignore.
	duc, err := dockerui.NewClient(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create dockerui client")
	}

	cacheImports, err := parseCacheOptions(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse cache import options")
	}

	// 4. Determine target platforms for the build.
	targetPlatforms := []*ocispecs.Platform{nil} // Default to BuildKit daemon's platform if none specified.
	if platformStr, exists := opts[keyTargetPlatform]; exists && platformStr != "" {
		parsedPlatforms, parseErr := parsePlatforms(platformStr)
		if parseErr != nil {
			return nil, errors.Wrapf(parseErr, "failed to parse target platforms from option: %s", platformStr)
		}
		targetPlatforms = parsedPlatforms
	}

	isMultiPlatform := len(targetPlatforms) > 1
	exportPlatforms := &exptypes.Platforms{
		Platforms: make([]exptypes.Platform, len(targetPlatforms)),
	}
	finalResult := gatewayclient.NewResult()
	eg, egCtx := errgroup.WithContext(ctx) // Use errgroup's context for concurrent operations.

	// 5. For each target platform, convert the generated Dockerfile content to LLB and solve.
	for i, tp := range targetPlatforms {
		currentIndex := i
		currentTargetPlatform := tp

		eg.Go(func() (err error) {
			// Prepare options for dockerfile2llb. This is where build args and .dockerignore handling
			// are implicitly configured for the standard converter.
			convertOpt := dockerfile2llb.ConvertOpt{
				Config: duc.Config, // Contains parsed BuildArgs, Labels, etc., from global opts.
				Client: duc,        // Allows dockerfile2llb to access MainContext (with .dockerignore).
				// MainContext: nil, // Setting to nil ensures dockerfile2llb uses Client.MainContext().
				TargetPlatform: currentTargetPlatform, // Platform for this specific LLB conversion.
				MetaResolver:   c,                     // For resolving image names.
				LLBCaps:        &buildOpts.LLBCaps,    // BuildKit features/capabilities.
			}

			// buildImage encapsulates Dockerfile2LLB conversion and the subsequent Solve.
			builtImageResult, buildErr := buildImage(egCtx, c, dockerfileContent, convertOpt, cacheImports, isMultiPlatform)
			if buildErr != nil {
				return errors.Wrapf(buildErr, "failed to build image for platform %v", currentTargetPlatform)
			}

			builtImageResult.AddToClientResult(finalResult)
			exportPlatforms.Platforms[currentIndex] = builtImageResult.ExportPlatform
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err // Return the first error encountered in the goroutines.
	}

	// If multi-platform, add platform metadata to the result.
	if isMultiPlatform {
		dt, err := json.Marshal(exportPlatforms)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal multi-platform export metadata")
		}
		finalResult.AddMeta(exptypes.ExporterPlatformsKey, dt)
	}

	return finalResult, nil
}

// parseCacheOptions extracts cache import configurations from the build options.
func parseCacheOptions(opts map[string]string) ([]gatewayclient.CacheOptionsEntry, error) {
	var cacheImports []gatewayclient.CacheOptionsEntry
	if cacheImportsStr, exists := opts[keyCacheImports]; exists && cacheImportsStr != "" {
		var cacheImportsUM []gatewayclient.CacheOptionsEntry
		if err := json.Unmarshal([]byte(cacheImportsStr), &cacheImportsUM); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal %s (%q)", keyCacheImports, cacheImportsStr)
		}
		cacheImports = append(cacheImports, cacheImportsUM...)
	}
	if cacheFromStr, exists := opts[keyCacheFrom]; exists && cacheFromStr != "" { // Legacy cache-from
		legacyEntries := strings.Split(cacheFromStr, ",")
		for _, s := range legacyEntries {
			trimmedEntry := strings.TrimSpace(s)
			if trimmedEntry != "" {
				im := gatewayclient.CacheOptionsEntry{
					Type:  "registry",
					Attrs: map[string]string{"ref": trimmedEntry},
				}
				cacheImports = append(cacheImports, im)
			}
		}
	}
	return cacheImports, nil
}

// buildResult holds information about a single platform's build result.
type buildResult struct {
	Reference      gatewayclient.Reference
	ImageConfig    []byte
	BuildInfo      []byte // Not currently populated. For future SBOM/metadata.
	Platform       *ocispecs.Platform
	MultiPlatform  bool
	ExportPlatform exptypes.Platform
}

// AddToClientResult merges a single platform's build result into the final gateway client result.
func (br *buildResult) AddToClientResult(cr *gatewayclient.Result) {
	if br.MultiPlatform {
		cr.AddMeta(fmt.Sprintf("%s/%s", exptypes.ExporterImageConfigKey, br.ExportPlatform.ID), br.ImageConfig)
		cr.AddRef(br.ExportPlatform.ID, br.Reference)
	} else {
		cr.AddMeta(exptypes.ExporterImageConfigKey, br.ImageConfig)
		cr.SetRef(br.Reference)
	}
}

// buildImage converts Dockerfile content to LLB and solves it for a specific platform.
func buildImage(ctx context.Context, c gatewayclient.Client, dockerfileContent string, convertOpts dockerfile2llb.ConvertOpt, cacheImports []gatewayclient.CacheOptionsEntry, isMultiPlatformBuild bool) (*buildResult, error) {
	result := buildResult{
		Platform:      convertOpts.TargetPlatform,
		MultiPlatform: isMultiPlatformBuild,
	}

	// Convert the (generated) Dockerfile content to LLB state.
	// Build args and .dockerignore (via convertOpts.Client.MainContext) are handled here.
	state, image, _, _, err := dockerfile2llb.Dockerfile2LLB(ctx, []byte(dockerfileContent), convertOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert Dockerfile content to LLB state")
	}

	result.ImageConfig, err = json.Marshal(image)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal image config to JSON")
	}

	result.BuildInfo = []byte{} // SBOM targets (4th return from Dockerfile2LLB) are not processed here.

	// Marshal the LLB state to its protobuf definition for solving.
	def, err := state.Marshal(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LLB state to definition")
	}

	// Solve the LLB definition with BuildKit.
	solveRequest := gatewayclient.SolveRequest{
		Definition:   def.ToPB(),
		CacheImports: cacheImports,
	}
	res, err := c.Solve(ctx, solveRequest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to solve LLB definition")
	}

	result.Reference, err = res.SingleRef()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get single reference from solve result")
	}

	// Prepare platform information for export.
	result.ExportPlatform = exptypes.Platform{Platform: platforms.DefaultSpec()} // Default
	if result.Platform != nil {
		result.ExportPlatform.Platform = *result.Platform // Override if specific
	}
	result.ExportPlatform.ID = platforms.Format(result.ExportPlatform.Platform)

	return &result, nil
}

// readMopyConfig loads the Mopyfile from the main build context.
func readMopyConfig(ctx context.Context, c gatewayclient.Client) (*config.Config, error) {
	opts := c.BuildOpts().Opts
	filename := opts[keyConfigPath] // Get Mopyfile path from --opt filename=...
	if filename == "" {
		filename = defaultDockerfileName
	}

	internalName := "load mopy definition"
	if filename != defaultDockerfileName {
		internalName += " from " + filename
	}

	// Define the LLB source for the Mopyfile, expecting it in the main build context.
	src := llb.Local(
		localNameContext,                        // Load from the main build context (e.g., "context")
		llb.IncludePatterns([]string{filename}), // Target the specific Mopyfile.
		llb.SessionID(c.BuildOpts().SessionID),
		llb.SharedKeyHint(defaultDockerfileName), // Cache hint.
		dockerui.WithInternalName(internalName),  // Internal name for BuildKit logs.
	)

	def, err := src.Marshal(ctx) // Use the passed-in context for marshalling.
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal local source for Mopyfile (%s)", filename)
	}

	res, err := c.Solve(ctx, gatewayclient.SolveRequest{Definition: def.ToPB()})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to solve for Mopyfile source (%s)", filename)
	}

	ref, err := res.SingleRef()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get single reference for Mopyfile source solve (%s)", filename)
	}

	mopyfileYaml, err := ref.ReadFile(ctx, gatewayclient.ReadRequest{Filename: filename})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read Mopyfile content: %s", filename)
	}

	cfg, err := config.NewFromBytes(mopyfileYaml)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse Mopyfile YAML content")
	}

	return cfg, nil
}

// parsePlatforms converts a comma-separated string of platform specs into a slice of *ocispecs.Platform.
func parsePlatforms(v string) ([]*ocispecs.Platform, error) {
	var pp []*ocispecs.Platform
	for _, platformStr := range strings.Split(v, ",") {
		trimmedPlatformStr := strings.TrimSpace(platformStr)
		if trimmedPlatformStr == "" {
			continue
		}
		p, err := platforms.Parse(trimmedPlatformStr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse platform component %q from %q", trimmedPlatformStr, v)
		}
		normalizedP := platforms.Normalize(p)
		pp = append(pp, &normalizedP)
	}
	if len(pp) == 0 {
		return []*ocispecs.Platform{nil}, nil // Represents the default platform.
	}
	return pp, nil
}
