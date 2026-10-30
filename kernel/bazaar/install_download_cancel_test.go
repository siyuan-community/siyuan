// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package bazaar

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
)

// useBazaarReleaseServer 让 githubClient 指向测试服务器：owner/repo 最新 Release 的 package.zip 由 download 提供。
func useBazaarReleaseServer(t *testing.T, download http.HandlerFunc) {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/repos/owner/repo/releases/latest", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(writer, `{"assets":[{"name":"package.zip","browser_download_url":%q}]}`, server.URL+"/package.zip")
	})
	mux.HandleFunc("/package.zip", download)
	baseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	oldClient := githubClient
	t.Cleanup(func() { githubClient = oldClient })
	githubClient = github.NewClient(server.Client())
	githubClient.BaseURL = baseURL
}

func TestOnlineInstallCancellationDuringDownload(t *testing.T) {
	for _, knownLength := range []bool{true, false} {
		t.Run(fmt.Sprint(knownLength), func(t *testing.T) {
			_, target := useInstallRevisionFixture(t)
			started, disconnected := make(chan struct{}), make(chan struct{})
			useBazaarReleaseServer(t, func(writer http.ResponseWriter, request *http.Request) {
				if knownLength {
					writer.Header().Set("Content-Length", "1000000")
				}
				_, _ = writer.Write([]byte("PK"))
				writer.(http.Flusher).Flush()
				close(started)
				<-request.Context().Done()
				close(disconnected)
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			options := installRevisionOptions(target)
			options.Context = ctx
			done := make(chan error, 1)
			go func() {
				_, err := InstallPackageWithOptions("https://github.com/owner/repo", "hash", "", target, "", "plugins", "sample", true, options)
				done <- err
			}()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("download did not start")
			}
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("download cancellation = %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("canceled download kept waiting")
			}
			select {
			case <-disconnected:
			case <-time.After(5 * time.Second):
				t.Fatal("cancellation did not close the HTTP request")
			}
			if data, err := os.ReadFile(filepath.Join(target, "index.js")); err != nil || string(data) != "old" {
				t.Fatal("canceled download replaced the installed package", err)
			}
		})
	}
}

func TestOnlineDownloadsHaveIndependentCancellation(t *testing.T) {
	started, release := make(chan struct{}, 2), make(chan struct{})
	var releaseOnce sync.Once
	releaseDownloads := func() { releaseOnce.Do(func() { close(release) }) }
	useBazaarReleaseServer(t, func(writer http.ResponseWriter, request *http.Request) {
		started <- struct{}{}
		select {
		case <-release:
			_, _ = writer.Write([]byte("package"))
		case <-request.Context().Done():
		}
	})
	defer releaseDownloads()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	canceled, completed := make(chan error, 1), make(chan error, 1)
	go func() {
		_, err := downloadBazaarPackageContext(ctx, "https://github.com/owner/repo@hash", false)
		canceled <- err
	}()
	go func() {
		data, err := downloadBazaarPackageContext(context.Background(), "https://github.com/owner/repo@hash", false)
		if err == nil && string(data) != "package" {
			err = fmt.Errorf("unexpected package: %q", data)
		}
		completed <- err
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("downloads were unexpectedly shared")
		}
	}
	cancel()
	select {
	case err := <-canceled:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("download did not cancel")
	}
	releaseDownloads()
	select {
	case err := <-completed:
		if err != nil {
			t.Fatal("canceling one download affected another", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("independent download did not finish")
	}
}

func TestOnlineInstallCancellationBeforeReplacement(t *testing.T) {
	_, target := useInstallRevisionFixture(t)
	data := buildInstallPackageArchive(t, map[string]string{
		"plugin.json": `{"name":"sample","version":"2.0.0"}`,
		"index.js":    "new",
	})
	useBazaarReleaseServer(t, func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(data)
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	options := installRevisionOptions(target)
	options.Context = ctx
	options.BeforeReplace = func() error { cancel(); return nil }
	_, err := InstallPackageWithOptions("https://github.com/owner/repo", "hash", "", target, "", "plugins", "sample", true, options)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("replacement cancellation = %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(target, "index.js")); err != nil || string(data) != "old" {
		t.Fatal("canceled installation replaced the package", err)
	}
	entries, err := os.ReadDir(filepath.Dir(target))
	if err != nil || len(entries) != 1 {
		t.Fatal("canceled installation retained staging files", entries, err)
	}
}
