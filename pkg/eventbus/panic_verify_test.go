package eventbus_test

// 验证：订阅者 panic 是否导致进程崩溃（修复前应为崩溃，修复后不应崩溃）。
// 必须在子进程中运行：go test 框架会 recover 掉测试自身的 panic，
// 掩盖掉真实行为，无法验证进程级影响。
import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"go-iot/pkg/eventbus"
)

func TestSubscriberPanicCrashesProcess(t *testing.T) {
	if os.Getenv("PANIC_VERIFY_CHILD") == "1" {
		// 子进程：订阅一个必 panic 的回调并发布
		eventbus.Subscribe("/device/*/*/*", func(eventbus.Message) { panic("boom") })
		eventbus.Publish("/device/p1/d1/property", &eventbus.PropertiesMessage{})
		// 若 publish 内部有 recover，能走到这里；否则进程在此前崩溃
		time.Sleep(200 * time.Millisecond)
		os.Exit(0)
	}

	// 父进程：启动子进程并观察退出码
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run", "TestSubscriberPanicCrashesProcess$")
	cmd.Env = append(os.Environ(), "PANIC_VERIFY_CHILD=1")
	cmd.Dir = filepath.Dir(exe)
	err = cmd.Run()

	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	if code == 0 {
		t.Log("PASS: 订阅者 panic 未导致进程崩溃（有 recover 保护）")
	} else {
		t.Logf("FAIL: 订阅者 panic 导致进程崩溃，退出码=%d", code)
	}
}
