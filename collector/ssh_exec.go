package collector

import (
	"fmt"
	"log/slog"
	"os"

	"golang.org/x/crypto/ssh"
)

func CollectBySsh(hostName string, command string, output chan string) {
	targetHost := GetConfig(hostName)
	jumpHost := GetConfig(targetHost.Proxy)
	if jumpHost.Proxy != "" {
		output <- SshMultiJump(hostName, command)
	} else if targetHost.Proxy != "" {
		output <- SshSingleJump(hostName, command)
	} else {
		output <- Ssh(hostName, command)
	}
}

func GetSshConfig(hostConfig HostConfig) *ssh.ClientConfig {
	var jumpPrivateKeyBytes []byte
	var err error
	var jumpPrivateKey ssh.Signer

	if hostConfig.Key != "" {
		jumpPrivateKeyBytes, err = os.ReadFile(hostConfig.Key)
		if err != nil {
			slog.Error("Failed to read private key file: %v", err)
			return nil
		}

		if hostConfig.Passphrase != "" {
			jumpPrivateKey, err = ssh.ParsePrivateKeyWithPassphrase(jumpPrivateKeyBytes, []byte(hostConfig.Passphrase))
			if err != nil {
				slog.Error("Failed to parse private key: %v", err)
				return nil
			}
		} else {
			jumpPrivateKey, err = ssh.ParsePrivateKey(jumpPrivateKeyBytes)
			if err != nil {
				slog.Error("private key error: %v", err)
				return nil
			}
		}
	}

	// Set up SSH client configuration for the jump host.
	sshConfig := &ssh.ClientConfig{
		User: hostConfig.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(hostConfig.Password),
			ssh.PublicKeys(jumpPrivateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshConfig.SetDefaults()
	return sshConfig
}

func Ssh(hostName string, command string) string {
	targetHost := GetConfig(hostName)
	targetConfig := GetSshConfig(targetHost)

	// Establish a connection to the host.
	targetConn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", targetHost.Ip, targetHost.Port), targetConfig)
	if err != nil {
		slog.Error("Failed to connect to target host: %v", err)
		return "# " + err.Error()
	}
	defer targetConn.Close()

	targetSession, err := targetConn.NewSession()
	if err != nil {
		slog.Error("Failed to create SSH session to the target host: %v", err)
		return "# " + err.Error()
	}
	defer targetSession.Close()

	// Run the command
	output, err := targetSession.CombinedOutput(command)
	if err != nil {
		slog.Error("Failed to run command: ", err)
		return "# " + err.Error()
	}

	return string(output)
}

func SshSingleJump(hostName string, command string) string {
	targetHost := GetConfig(hostName)
	jumpHost := GetConfig(targetHost.Proxy)

	jumpConfig := GetSshConfig(jumpHost)
	targetConfig := GetSshConfig(targetHost)

	// Establish a connection to the jump host.
	jumpConn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", jumpHost.Ip, jumpHost.Port), jumpConfig)
	if err != nil {
		slog.Error("Failed to connect to jump host: %v", err)
		return "# " + err.Error()
	}
	defer jumpConn.Close()

	// Establish a nested SSH connection to the target host through the jump host.
	targetConn, err := jumpConn.Dial("tcp", fmt.Sprintf("%s:%s", targetHost.Ip, targetHost.Port))
	if err != nil {
		slog.Error("Failed to establish a nested connection to the target host: %v", err)
		return "# " + err.Error()
	}
	defer targetConn.Close()

	ncc, chans, reqs, err := ssh.NewClientConn(targetConn, targetHost.Ip, targetConfig)
	if err != nil {
		slog.Error("Failed to create SSH client connection to the target host: %v", err)
		return "# " + err.Error()
	}
	defer ncc.Close()

	targetClient := ssh.NewClient(ncc, chans, reqs)
	defer targetClient.Close()

	targetSession, err := targetClient.NewSession()
	if err != nil {
		slog.Error("Failed to create SSH session to the target host: %v", err)
		return "# " + err.Error()
	}
	defer targetSession.Close()

	// Run the command
	output, err := targetSession.CombinedOutput(command)
	if err != nil {
		slog.Error("Failed to run command: ", err)
		return "# " + err.Error()
	}

	return string(output)
}

func SshMultiJump(hostName string, command string) string {
	targetHost := GetConfig(hostName)
	jumpHost2 := GetConfig(targetHost.Proxy)
	jumpHost1 := GetConfig(jumpHost2.Proxy)

	jumpConfig1 := GetSshConfig(jumpHost1)
	jumpConfig2 := GetSshConfig(jumpHost2)
	targetConfig := GetSshConfig(targetHost)

	// Establish a connection to the first jump host.
	jumpConn1, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", jumpHost1.Ip, jumpHost1.Port), jumpConfig1)
	if err != nil {
		slog.Error("Failed to connect to jump host 1: %v", err)
		return "# " + err.Error()
	}
	defer jumpConn1.Close()

	// Establish a connection to the second jump host.
	jumpConn2, err := jumpConn1.Dial("tcp", fmt.Sprintf("%s:%s", jumpHost2.Ip, jumpHost2.Port))
	if err != nil {
		slog.Error("Failed to connect to jump host 1: %v", err)
		return "# " + err.Error()
	}
	defer jumpConn2.Close()
	jumpncc2, chans, reqs, err := ssh.NewClientConn(jumpConn2, jumpHost2.Ip, jumpConfig2)
	if err != nil {
		slog.Error("Failed to create SSH client connection to the target host: %v", err)
		return "# " + err.Error()
	}
	defer jumpncc2.Close()
	jumpClient2 := ssh.NewClient(jumpncc2, chans, reqs)
	defer jumpClient2.Close()

	// Establish a nested SSH connection to the target host through the jump host.
	targetConn, err := jumpClient2.Dial("tcp", fmt.Sprintf("%s:%s", targetHost.Ip, targetHost.Port))
	if err != nil {
		slog.Error("Failed to establish a nested connection to the target host: %v", err)
		return "# " + err.Error()
	}
	defer targetConn.Close()

	ncc, chans, reqs, err := ssh.NewClientConn(targetConn, targetHost.Ip, targetConfig)
	if err != nil {
		slog.Error("Failed to create SSH client connection to the target host: %v", err)
		return "# " + err.Error()
	}
	defer ncc.Close()

	targetClient := ssh.NewClient(ncc, chans, reqs)
	defer targetClient.Close()

	targetSession, err := targetClient.NewSession()
	if err != nil {
		slog.Error("Failed to create SSH session to the target host: %v", err)
		return "# " + err.Error()
	}
	defer targetSession.Close()

	// Run the command
	output, err := targetSession.CombinedOutput(command)
	if err != nil {
		slog.Error("Failed to run command: ", err)
		return "# " + err.Error()
	}

	return string(output)
}
