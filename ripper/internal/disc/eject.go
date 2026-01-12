package disc

import "os/exec"

func Eject(drive string) error {
	command := "(New-Object -COMObject Shell.Application).Namespace(17).ParseName('" + drive + "').InvokeVerb('Eject')"
	return exec.Command("powershell", "-Command", command).Run()
}
