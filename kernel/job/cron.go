// SiYuan - Refactor your thinking
// Copyright (c) 2020-present, b3log.org
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package job

import (
	"time"

	"github.com/siyuan-community/siyuan/kernel/model"
	"github.com/siyuan-community/siyuan/kernel/sql"
	"github.com/siyuan-community/siyuan/kernel/task"
	"github.com/siyuan-community/siyuan/kernel/util"
	"github.com/siyuan-note/logging"
)

func StartCron() {
	go every(100*time.Millisecond, task.ExecTaskJob)
	go every(100*time.Millisecond, task.ExecAsyncTaskJob)
	go every(5*time.Second, task.StatusJob)
	go every(5*time.Second, model.SyncDataJob)
	go every(2*time.Hour, model.StatJob)
	go every(6*time.Hour, util.RefreshRhyResultJob, "RefreshRhyResultJob")
	go every(2*time.Hour, model.RefreshCheckJob)
	go every(3*time.Second, model.FlushUpdateRefTextRenameDocJob)
	go every(util.SQLFlushInterval, sql.FlushTxJob)
	go every(util.SQLFlushInterval, sql.FlushHistoryTxJob)
	go every(util.SQLFlushInterval, sql.FlushAssetContentTxJob)
	go every(10*time.Minute, model.IndexEmbedBlockJob)
	go every(10*time.Minute, model.CacheVirtualBlockRefJob)
	go every(30*time.Second, model.OCRAssetsJob)
	go every(30*time.Second, model.FlushAssetsTextsJob)
	go every(30*time.Second, model.HookDesktopUIProcJob)
	go every(24*time.Hour, model.AutoPurgeRepoJob)
	go every(30*time.Minute, model.AutoCheckMicrosoftDefenderJob)

	// TODO: 移除旧方案 https://github.com/siyuan-note/siyuan/issues/14414 实现新的刷新机制
	//go every(3*time.Second, model.WatchLocalShorthands)
}

func every(interval time.Duration, f func(), name ...string) {
	util.RandomSleep(50, 200)

	// 启动后立即执行一次
	func() {
		defer logging.Recover()
		f()
		if 0 < len(name) {
			logging.LogInfof("cron job [%s] executed", name)
		}
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		func() {
			defer logging.Recover()
			f()
			if 0 < len(name) {
				logging.LogInfof("cron job [%s] executed", name)
			}
		}()
	}
}
