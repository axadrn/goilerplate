package desktop

import (
	"reflect"
	"testing"
)

func TestBrowserCommandUsesSupportedPlatformLauncher(t *testing.T) {
	tests := []struct {
		goos string
		want command
	}{
		{goos: "darwin", want: command{name: "open", args: []string{"https://github.com/login/device"}}},
		{goos: "linux", want: command{name: "xdg-open", args: []string{"https://github.com/login/device"}}},
	}
	for _, test := range tests {
		t.Run(test.goos, func(t *testing.T) {
			got, err := browserCommand(test.goos, "https://github.com/login/device")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("browser command = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestBrowserCommandRejectsEmptyAndUnsupportedInputs(t *testing.T) {
	if _, err := browserCommand("linux", " "); err == nil {
		t.Fatal("empty browser address was accepted")
	}
	if _, err := browserCommand("plan9", "https://github.com/login/device"); err == nil {
		t.Fatal("unsupported platform was accepted")
	}
}

func TestClipboardCommandsCoverSupportedPlatforms(t *testing.T) {
	if got := clipboardCommands("darwin"); !reflect.DeepEqual(got, []command{{name: "pbcopy"}}) {
		t.Fatalf("darwin clipboard commands = %#v", got)
	}
	if got := clipboardCommands("linux"); !reflect.DeepEqual(got, []command{
		{name: "wl-copy"},
		{name: "xclip", args: []string{"-selection", "clipboard"}},
		{name: "xsel", args: []string{"--clipboard", "--input"}},
		{name: "clip.exe"},
	}) {
		t.Fatalf("linux clipboard commands = %#v", got)
	}
	if got := clipboardCommands("plan9"); got != nil {
		t.Fatalf("unsupported clipboard commands = %#v", got)
	}
}
