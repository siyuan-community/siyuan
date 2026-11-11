// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/siyuan-community/siyuan/kernel/conf"
	"github.com/siyuan-community/siyuan/kernel/util"
)

func TestThirdPartySyncAddressValidation(t *testing.T) {
	for _, provider := range []int{conf.ProviderWebDAV, conf.ProviderLocal} {
		t.Run(conf.ProviderToStr(provider), func(t *testing.T) {
			prepareSyncDirectoryTest(t)
			Conf.Sync.Load().Provider, Conf.Sync.Load().Enabled = provider, true
			Conf.Sync.Load().WebDAV = &conf.WebDAV{Endpoint: "https://dav.example", Username: "old", Password: "secret"}
			Conf.Sync.Load().Local = &conf.Local{Endpoint: t.TempDir()}
			Conf.Save()
			configPath := filepath.Join(util.ConfDir, "conf.json")
			before, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			oldDAV, oldLocal := Conf.Sync.Load().WebDAV, Conf.Sync.Load().Local
			values := []string{"", " \t\n "}
			if provider == conf.ProviderWebDAV {
				values = append(values, "https://", "ftp://dav.example", "https://bad host/")
			}
			for _, value := range values {
				if provider == conf.ProviderWebDAV {
					err = SetSyncProviderWebDAV(&conf.WebDAV{Endpoint: value})
				} else {
					err = SetSyncProviderLocal(&conf.Local{Endpoint: value})
				}
				if err == nil || !Conf.Sync.Load().Enabled || Conf.Sync.Load().WebDAV != oldDAV || Conf.Sync.Load().Local != oldLocal {
					t.Fatalf("invalid %s address %q changed configuration: %v", conf.ProviderToStr(provider), value, err)
				}
				after, readErr := os.ReadFile(configPath)
				if readErr != nil || !bytes.Equal(before, after) {
					t.Fatal("rejected address changed persisted configuration")
				}
			}
			Conf.Sync.Load().WebDAV, Conf.Sync.Load().Local = &conf.WebDAV{}, &conf.Local{}
			if _, err = newCloudRepositoryWithAssetSourceLocked(); err == nil {
				t.Fatal("cloud operations accepted an existing empty address")
			}
			if _, err = newSyncRepositoryWithAssetSourceLocked(); err == nil || !Conf.Sync.Load().Enabled {
				t.Fatal("sync validation failed to preserve the enabled flag")
			}
			if _, err = newRepository(); err != nil {
				t.Fatalf("empty cloud configuration blocked local snapshots: %v", err)
			}
		})
	}
}

func TestThirdPartySyncAddressCompatibility(t *testing.T) {
	prepareSyncDirectoryTest(t)
	for _, endpoint := range []string{"https://dav.example/path", "http://localhost:9000/dav", "dav.example/path", "http(s)://dav.example/path"} {
		candidate := &conf.WebDAV{Endpoint: " " + endpoint + " ", Username: "", Password: ""}
		if err := SetSyncProviderWebDAV(candidate); err != nil {
			t.Fatalf("anonymous WebDAV address %q was rejected: %v", endpoint, err)
		}
		if Conf.Sync.Load().WebDAV.Endpoint != util.NormalizeEndpoint(endpoint) || Conf.Sync.Load().WebDAV.Username != "" || Conf.Sync.Load().WebDAV.Password != "" {
			t.Fatal("anonymous access or endpoint normalization changed")
		}
	}
	if err := SetSyncProviderLocal(&conf.Local{Endpoint: t.TempDir()}); err != nil {
		t.Fatalf("valid local directory was rejected: %v", err)
	}
	if err := SetSyncProviderWebDAV(nil); err == nil {
		t.Fatal("nil WebDAV configuration was accepted")
	}
	if err := SetSyncProviderLocal(nil); err == nil {
		t.Fatal("nil local configuration was accepted")
	}
}

func TestThirdPartySyncWithoutCloudUser(t *testing.T) {
	prepareSyncDirectoryTest(t)
	Conf.SetUser(nil)
	Conf.Sync.Load().Enabled = true
	Conf.Sync.Load().CloudName = "main"

	Conf.Sync.Load().Provider = conf.ProviderSiYuan
	if checkSync(false, false, true) {
		t.Fatal("SiYuan cloud sync succeeded without a logged-in user")
	}
	if err := checkAssetDownloadAccess(); err == nil {
		t.Fatal("SiYuan cloud asset download access succeeded without a logged-in user")
	}

	for _, provider := range []int{conf.ProviderS3, conf.ProviderWebDAV, conf.ProviderLocal} {
		Conf.Sync.Load().Provider = provider
		if !checkSync(false, false, true) {
			t.Fatalf("third-party provider %s sync rejected without a logged-in user", conf.ProviderToStr(provider))
		}
		if err := checkAssetDownloadAccess(); err != nil {
			t.Fatalf("third-party provider %s asset download access rejected without a logged-in user: %v", conf.ProviderToStr(provider), err)
		}
	}

	previousStatus := util.StatusBarCfg.Load()
	util.StatusBarCfg.Store(&util.StatusBar{MsgDataSyncDisabled: true})
	t.Cleanup(func() { util.StatusBarCfg.Store(previousStatus) })
	if err := os.MkdirAll(util.DataDir, 0755); err != nil {
		t.Fatal(err)
	}

	var s3Requested atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s3Requested.Store(true)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	Conf.Sync.Load().Provider = conf.ProviderS3
	Conf.Sync.Load().S3.Endpoint = server.URL
	_ = syncData(false, true)
	if !s3Requested.Load() {
		t.Fatal("S3 syncData did not reach the S3 endpoint when Conf.GetUser() == nil")
	}
}
