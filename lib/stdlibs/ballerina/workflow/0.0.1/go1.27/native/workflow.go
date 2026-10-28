// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package native

import (
	"strings"
	"time"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "workflow"
)

func initWorkflowModule(rt *runtime.Runtime) {
	eng := newEngine(rt)
	register := func(name string, fn extern.NativeFunc) {
		runtime.RegisterExternFunction(rt, orgName, moduleName, name, fn)
	}
	register("initModule", noop)
	register("initInMemoryProgramNative", noop)
	register("stopWorkflowRuntime", noop)
	register("initProgramNative", func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
		return values.NewErrorWithMessage("only the IN_MEMORY mode is supported"), nil
	})
	register("runNative", eng.run)
	register("sendDataNative", eng.sendData)
	register("getResult", eng.getResult)
	register("waitForResult", eng.waitForResult)
	register("getStatus", eng.getStatus)
	register("Context.$remote$callActivity", eng.callActivity)
	for _, name := range []string{"runWithIdNative", "getPendingAgentEvents", "completeHumanTaskNative",
		"Context.$remote$await", "Context.$remote$awaitHumanTask", "Context.$remote$runChildWorkflow",
		"Context.$remote$getChildWorkflowResult", "Context.$remote$waitForChildWorkflow",
		"Context.$remote$callWorkflow", "Context.$remote$sendDataToChildWorkflow"} {
		register(name, unsupported(name))
	}
	register("lastHumanTaskCompletionNative", noop)
	register("lastReviewDecisionNative", noop)
	register("isReplayingNative", func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
		return false, nil
	})
	register("getWorkflowIdNative", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return args[0].(*instance).id, nil
	})
	register("getWorkflowTypeNative", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return args[0].(*instance).workflowType, nil
	})
	register("currentTimeMillisContextNative", func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
		return rt.Platform().Time.Now().UnixMilli(), nil
	})
	register("sleepContextNative", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		time.Sleep(time.Duration(args[1].(int64)) * time.Millisecond)
		return nil, nil
	})
}

func noop(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
	return nil, nil
}

// unsupported returns an error-producing extern for an API the in-memory engine does not implement yet.
func unsupported(name string) extern.NativeFunc {
	return func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
		return values.NewErrorWithMessage("workflow API not supported by the in-memory engine yet: " + name), nil
	}
}

// splitLookupKey splits a function lookup key of the form "org/module:name".
func splitLookupKey(key string) (org, module, name string) {
	slash := strings.Index(key, "/")
	colon := strings.LastIndex(key, ":")
	return key[:slash], key[slash+1 : colon], key[colon+1:]
}

func init() {
	runtime.RegisterModuleInitializer(initWorkflowModule)
}
