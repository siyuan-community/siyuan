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

package api

import (
	"net/http"

	"github.com/88250/gulu"
	"github.com/gin-gonic/gin"
	"github.com/siyuan-community/siyuan/kernel/model"
	"github.com/siyuan-community/siyuan/kernel/util"
)

func getDocOutline(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	if nil == arg["id"] {
		return
	}

	preview := false
	if previewArg := arg["preview"]; nil != previewArg {
		preview = previewArg.(bool)
	}

	rootID := arg["id"].(string)
	headings, err := model.Outline(rootID, preview)
	if err != nil {
		ret.Code = 1
		ret.Msg = err.Error()
		return
	}
	ret.Data = headings
}
