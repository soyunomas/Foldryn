package notifier

import "os/exec"

type Notifier struct{ Enabled bool }

func (n Notifier) Send(title, body string) {
	if !n.Enabled {
		return
	}
	if _, err := exec.LookPath("notify-send"); err != nil {
		return
	}
	_ = exec.Command("notify-send", title, body).Start()
}
