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

package sql

import (
	"database/sql"
	"path/filepath"
	"strings"

	"github.com/88250/lute/ast"
	"github.com/siyuan-community/siyuan/kernel/treenode"
	"github.com/siyuan-community/siyuan/kernel/util"
	"github.com/siyuan-note/filelock"
	"github.com/siyuan-note/logging"
)

type Asset struct {
	ID      string
	BlockID string
	RootID  string
	Box     string
	DocPath string
	Path    string
	Name    string
	Title   string
	Hash    string
}

func docTagSpans(n *ast.Node) (ret []*Span) {
	if tagsVal := n.IALAttr("tags"); "" != tagsVal {
		tags := strings.Split(tagsVal, ",")
		for _, tag := range tags {
			escaped := util.EscapeHTML(tag)
			markdown := "#" + escaped + "#"
			span := &Span{
				ID:       ast.NewNodeID(),
				BlockID:  n.ID,
				RootID:   n.ID,
				Box:      n.Box,
				Path:     n.Path,
				Content:  tag,
				Markdown: markdown,
				Type:     "tag",
				IAL:      "",
			}
			ret = append(ret, span)
		}
	}
	return
}

func docTitleImgAsset(root *ast.Node, boxLocalPath, docDirLocalPath string) *Asset {
	if p := treenode.GetDocTitleImgPath(root); "" != p {
		if !util.IsAssetLinkDest([]byte(p)) {
			return nil
		}

		var hash string
		var err error
		if lp := assetLocalPath(p, boxLocalPath, docDirLocalPath); "" != lp {
			hash, err = util.GetEtag(lp)
			if err != nil {
				logging.LogErrorf("calc asset [%s] hash failed: %s", lp, err)
				return nil
			}
		}

		name, _ := util.LastID(p)
		asset := &Asset{
			ID:      ast.NewNodeID(),
			BlockID: root.ID,
			RootID:  root.ID,
			Box:     root.Box,
			DocPath: root.Path,
			Path:    p,
			Name:    name,
			Title:   "title-img",
			Hash:    hash,
		}
		return asset
	}
	return nil
}

func deleteAssetsByHashes(tx *sql.Tx, hashes []string) (err error) {
	sqlStmt := "DELETE FROM assets WHERE hash IN ('" + strings.Join(hashes, "','") + "') OR hash = ''"
	err = execStmtTx(tx, sqlStmt)
	return
}

func QueryAssetByHash(hash string) (ret *Asset) {
	sqlStmt := "SELECT * FROM assets WHERE hash = ?"
	row := queryRow(sqlStmt, hash)
	var asset Asset
	if err := row.Scan(&asset.ID, &asset.BlockID, &asset.RootID, &asset.Box, &asset.DocPath, &asset.Path, &asset.Name, &asset.Title, &asset.Hash); err != nil {
		if sql.ErrNoRows != err {
			logging.LogErrorf("query scan field failed: %s", err)
		}
		return
	}
	ret = &asset
	return
}

func scanAssetRows(rows *sql.Rows) (ret *Asset) {
	var asset Asset
	if err := rows.Scan(&asset.ID, &asset.BlockID, &asset.RootID, &asset.Box, &asset.DocPath, &asset.Path, &asset.Name, &asset.Title, &asset.Hash); err != nil {
		logging.LogErrorf("query scan field failed: %s", err)
		return
	}
	ret = &asset
	return
}

func assetLocalPath(linkDest, boxLocalPath, docDirLocalPath string) (ret string) {
	ret = filepath.Join(docDirLocalPath, linkDest)
	if filelock.IsExist(ret) {
		return
	}

	ret = filepath.Join(boxLocalPath, linkDest)
	if filelock.IsExist(ret) {
		return
	}

	ret = filepath.Join(util.DataDir, linkDest)
	if filelock.IsExist(ret) {
		return
	}
	return ""
}
