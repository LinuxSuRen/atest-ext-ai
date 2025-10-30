package plugin

import (
	"runtime/debug"
	"strings"
)

const (
	defaultPluginVersion       = "v1.0.0"
	defaultAPITestingVersion   = "v0.0.19"
	apiTestingModulePath       = "github.com/linuxsuren/api-testing"
	pluginModulePath           = "github.com/linuxsuren/atest-ext-ai"
	developmentVersionSentinel = "(devel)"
)

// buildPluginVersion can be overridden at link time via -ldflags.
var buildPluginVersion string

// buildAPITestingVersion allows overriding the detected api-testing version via -ldflags.
var buildAPITestingVersion string

func detectPluginVersion() string {
	if v := normalizeVersion(buildPluginVersion); v != "" {
		return v
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Path == pluginModulePath {
			if v := normalizeVersion(info.Main.Version); v != "" {
				return v
			}
		}
		for _, setting := range info.Settings {
			if setting.Key == "vcs.tag" {
				if v := normalizeVersion(setting.Value); v != "" {
					return v
				}
			}
		}
	}

	return defaultPluginVersion
}

func detectAPITestingVersion() string {
	if v := normalizeVersion(buildAPITestingVersion); v != "" {
		return v
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == apiTestingModulePath {
				if v := normalizeVersion(dep.Version); v != "" {
					return v
				}
			}
		}
	}

	return defaultAPITestingVersion
}

func normalizeVersion(v string) string {
	if v == "" || v == developmentVersionSentinel {
		return ""
	}
	if strings.HasPrefix(v, "v") || strings.HasPrefix(v, "V") {
		return v
	}
	return "v" + v
}
