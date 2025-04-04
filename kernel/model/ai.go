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

package model

import (
	"bytes"
	"strings"

	"github.com/88250/lute/ast"
	"github.com/88250/lute/parse"
	"github.com/sashabaranov/go-openai"
	"github.com/siyuan-community/siyuan/kernel/treenode"
	"github.com/siyuan-community/siyuan/kernel/util"
)

func ChatGPT(msg string) (ret string) {
	if !isOpenAIAPIEnabled() {
		return
	}

	return chatGPT(msg, false)
}

func ChatGPTWithAction(ids []string, action string) (ret string) {
	if !isOpenAIAPIEnabled() {
		return
	}

	if "Clear context" == action {
		// AI clear context action https://github.com/siyuan-note/siyuan/issues/10255
		cachedContextMsg = nil
		return
	}

	msg := getBlocksContent(ids)
	ret = chatGPTWithAction(msg, action, false)
	return
}

var cachedContextMsg []string

func chatGPT(msg string, cloud bool) (ret string) {
	if "Clear context" == strings.TrimSpace(msg) {
		// AI clear context action https://github.com/siyuan-note/siyuan/issues/10255
		cachedContextMsg = nil
		return
	}

	ret, retCtxMsgs, err := chatGPTContinueWrite(msg, cachedContextMsg, cloud)
	if err != nil {
		return
	}
	cachedContextMsg = append(cachedContextMsg, retCtxMsgs...)
	return
}

func chatGPTWithAction(msg string, action string, cloud bool) (ret string) {
	action = strings.TrimSpace(action)
	if "" != action {
		msg = action + ":\n\n" + msg
	}
	ret, _, err := chatGPTContinueWrite(msg, nil, cloud)
	if err != nil {
		return
	}
	return
}

func chatGPTContinueWrite(msg string, contextMsgs []string, cloud bool) (ret string, retContextMsgs []string, err error) {
	util.PushEndlessProgress("Requesting...")
	defer util.ClearPushProgress(100)

	if Conf.AI.OpenAI.APIMaxContexts < len(contextMsgs) {
		contextMsgs = contextMsgs[len(contextMsgs)-Conf.AI.OpenAI.APIMaxContexts:]
	}

	var gpt GPT
	if cloud {
		gpt = &CloudGPT{}
	} else {
		gpt = &OpenAIGPT{c: util.NewOpenAIClient(Conf.AI.OpenAI.APIKey, Conf.AI.OpenAI.APIProxy, Conf.AI.OpenAI.APIBaseURL, Conf.AI.OpenAI.APIUserAgent, Conf.AI.OpenAI.APIVersion, Conf.AI.OpenAI.APIProvider)}
	}

	buf := &bytes.Buffer{}
	for i := 0; i < Conf.AI.OpenAI.APIMaxContexts; i++ {
		part, stop, chatErr := gpt.chat(msg, contextMsgs)
		buf.WriteString(part)

		if stop || nil != chatErr {
			break
		}

		util.PushEndlessProgress("Continue requesting...")
	}

	ret = buf.String()
	ret = strings.TrimSpace(ret)
	if "" != ret {
		retContextMsgs = append(retContextMsgs, msg, ret)
	}
	return
}

func isOpenAIAPIEnabled() bool {
	if "" == Conf.AI.OpenAI.APIKey {
		util.PushMsg(Conf.Language(193), 5000)
		return false
	}
	return true
}

func getBlocksContent(ids []string) string {
	var nodes []*ast.Node
	trees := map[string]*parse.Tree{}
	for _, id := range ids {
		bt := treenode.GetBlockTree(id)
		if nil == bt {
			continue
		}

		var tree *parse.Tree
		if tree = trees[bt.RootID]; nil == tree {
			tree, _ = LoadTreeByBlockID(bt.RootID)
			if nil == tree {
				continue
			}

			trees[bt.RootID] = tree
		}

		if node := treenode.GetNodeInTree(tree, id); nil != node {
			if ast.NodeDocument == node.Type {
				for child := node.FirstChild; nil != child; child = child.Next {
					nodes = append(nodes, child)
				}
			} else {
				nodes = append(nodes, node)
			}
		}
	}

	luteEngine := util.NewLute()
	buf := bytes.Buffer{}
	for _, node := range nodes {
		md := treenode.ExportNodeStdMd(node, luteEngine)
		buf.WriteString(md)
		buf.WriteString("\n\n")
	}
	return buf.String()
}

type GPT interface {
	chat(msg string, contextMsgs []string) (partRet string, stop bool, err error)
}

type OpenAIGPT struct {
	c *openai.Client
}

func (gpt *OpenAIGPT) chat(msg string, contextMsgs []string) (partRet string, stop bool, err error) {
	return util.ChatGPT(msg, contextMsgs, gpt.c, Conf.AI.OpenAI.APIModel, Conf.AI.OpenAI.APIMaxTokens, Conf.AI.OpenAI.APITemperature, Conf.AI.OpenAI.APITimeout)
}

type CloudGPT struct {
}

func (gpt *CloudGPT) chat(msg string, contextMsgs []string) (partRet string, stop bool, err error) {
	return CloudChatGPT(msg, contextMsgs)
}
