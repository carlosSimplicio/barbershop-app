package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/moby/patternmatcher"
	"github.com/moby/patternmatcher/ignorefile"
)

func TestDockerContextsExcludeEnvironmentFiles(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(source), "..", "..", "..")
	for _, application := range []string{"backend", "frontend"} {
		t.Run(application, func(t *testing.T) {
			file, err := os.Open(filepath.Join(root, application, ".dockerignore"))
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			patterns, err := ignorefile.ReadAll(file)
			if err != nil {
				t.Fatal(err)
			}
			matcher, err := patternmatcher.New(patterns)
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{".env", ".env.local", ".env.production", ".env.production.local", "nested/.env", "nested/.env.local"} {
				excluded, err := matcher.MatchesOrParentMatches(name)
				if err != nil || !excluded {
					t.Errorf("environment file %s is included in the build context: %v", name, err)
				}
			}
			excluded, err := matcher.MatchesOrParentMatches(".env.example")
			if err != nil || excluded {
				t.Errorf("non-sensitive example must remain available: %v", err)
			}
		})
	}
}
