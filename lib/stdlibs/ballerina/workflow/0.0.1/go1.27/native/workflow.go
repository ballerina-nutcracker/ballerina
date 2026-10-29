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
	register("initProgramNative", initProgram)
	register("runNative", eng.run)
	register("sendDataNative", eng.sendData)
	register("getResult", eng.getResult)
	register("waitForResult", eng.waitForResult)
	register("getStatus", eng.getStatus)
	register("Context.$remote$callActivity", eng.callActivity)
	register("isReplayingNative", isReplaying)
	register("getWorkflowIdNative", getWorkflowID)
	register("getWorkflowTypeNative", getWorkflowType)
	register("currentTimeMillisContextNative", currentTimeMillis)
	register("sleepContextNative", sleepContext)
	register("lastHumanTaskCompletionNative", noop)
	register("lastReviewDecisionNative", noop)
	// Not implemented yet. Those whose declared return admits an error answer one; `await`
	// returns a bare T, so it panics, and `awaitHumanTask` answers its HumanTaskFailedError.
	for _, name := range []string{"runWithIdNative", "getPendingAgentEvents", "completeHumanTaskNative",
		"Context.$remote$runChildWorkflow", "Context.$remote$getChildWorkflowResult",
		"Context.$remote$waitForChildWorkflow", "Context.$remote$callWorkflow",
		"Context.$remote$sendDataToChildWorkflow"} {
		register(name, unsupported(name))
	}
	register("Context.$remote$await", unsupportedPanic("Context.await"))
	register("Context.$remote$awaitHumanTask", unsupportedHumanTask)
}

func noop(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
	return nil, nil
}

func initProgram(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
	return values.NewErrorWithMessage("only the IN_MEMORY mode is supported"), nil
}

func isReplaying(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
	return false, nil
}

func getWorkflowID(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return args[0].(*instance).id, nil
}

func getWorkflowType(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return args[0].(*instance).workflowType, nil
}

func currentTimeMillis(ctx *extern.Context, _ []values.BalValue) (values.BalValue, error) {
	return ctx.Env.Platform.Time.Now().UnixMilli(), nil
}

// sleepContext waits on the PAL clock while yielding, as runtime:sleep does, so the strands
// sharing this thread keep running.
func sleepContext(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	clock := ctx.Env.Platform.Time
	deadline := clock.MonotonicNow() + time.Duration(args[1].(int64))*time.Millisecond
	for clock.MonotonicNow() < deadline {
		<-ctx.Yield()
	}
	return nil, nil
}

func unsupportedMessage(name string) string {
	return "workflow API not supported by the in-memory engine yet: " + name
}

// unsupported returns an error-producing extern for an API the in-memory engine does not implement yet.
func unsupported(name string) extern.NativeFunc {
	return func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
		return values.NewErrorWithMessage(unsupportedMessage(name)), nil
	}
}

// unsupportedPanic is for an API whose declared return type has no room for an error.
func unsupportedPanic(name string) extern.NativeFunc {
	return func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
		panic(values.NewErrorWithMessage(unsupportedMessage(name)))
	}
}

func unsupportedHumanTask(ctx *extern.Context, _ []values.BalValue) (values.BalValue, error) {
	return moduleError(ctx, "newHumanTaskFailedError", unsupportedMessage("Context.awaitHumanTask")), nil
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
