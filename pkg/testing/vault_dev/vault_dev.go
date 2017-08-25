package vault_dev

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	vault "github.com/hashicorp/vault/api"
)

type VaultDev struct {
	client       *vault.Client
	server       *exec.Cmd
	vaultRunning chan struct{}
	port         int
}

func New(port int) *VaultDev {
	return &VaultDev{
		port: port,
	}
}

func (v *VaultDev) Start() error {
	args := []string{
		"server",
		"-dev",
		"-dev-root-token-id=root-token",
		fmt.Sprintf("-dev-listen-address=127.0.0.1:%d", v.port),
	}

	logrus.Infof("starting vault: %#+v", args)

	v.server = exec.Command("vault", args...)

	err := v.server.Start()
	if err != nil {
		return err
	}

	// this channel will close once vault is stopped
	v.vaultRunning = make(chan struct{}, 0)

	go func() {
		err := v.server.Wait()
		if err != nil {
			logrus.Warn("vault stopped with error: ", err)

		} else {
			logrus.Info("vault stopped")
		}
		close(v.vaultRunning)
	}()

	v.client, err = vault.NewClient(&vault.Config{
		Address: fmt.Sprintf("http://127.0.0.1:%d", v.port),
	})
	if err != nil {
		return err
	}
	v.client.SetToken("root-token")

	tries := 30
	for {
		select {
		case _, open := <-v.vaultRunning:
			if !open {
				return fmt.Errorf("vault could not be started")
			}
		default:
		}

		_, err := v.client.Auth().Token().LookupSelf()
		if err == nil {
			break
		}
		if tries <= 1 {
			return fmt.Errorf("vault dev server couldn't be started in time")
		}
		tries -= 1
		time.Sleep(time.Second)
	}

	return nil
}

func (v *VaultDev) Stop() {
	if err := v.server.Process.Signal(syscall.SIGTERM); err != nil {
		logrus.Warn("killing vault dev server failed: ", err)
	}

	<-v.vaultRunning
}

func (v *VaultDev) Client() *vault.Client {
	return v.client
}
