package collector

import (
	"bytes"
	"log/slog"
	"os"
	"sync"

	"golang.org/x/crypto/ssh"
)

func (target *Target) SshRoute(command string) (*string, error) {
	jumpHostCount := len(target.JumpHosts)
	switch jumpHostCount {
	case 1:
		return SshOneJump(target, command)
	case 2:
		return SshTwoJump(target, command)
	default:
		return Ssh(target, command)
	}
}

func SshConfig(user string, private_key string, passphrase string, password string) *ssh.ClientConfig {
	passphrase, _ = Decrypt(passphrase)
	password, _ = Decrypt(password)
	var auth []ssh.AuthMethod

	if private_key != "" {
		privateKeyBytes, err := os.ReadFile(private_key)
		if err != nil {
			slog.Error("Failed to read private key file: %v", err)
			return nil
		}

		if passphrase != "" {
			privateKey, err := ssh.ParsePrivateKeyWithPassphrase(privateKeyBytes, []byte(passphrase))
			if err != nil {
				slog.Error("Failed to parse private key: %v", err)
				return nil
			}
			auth = []ssh.AuthMethod{ssh.PublicKeys(privateKey)}
		} else {
			privateKey, err := ssh.ParsePrivateKey(privateKeyBytes)
			if err != nil {
				slog.Error("private key error: %v", err)
				return nil
			}
			auth = []ssh.AuthMethod{ssh.PublicKeys(privateKey)}
		}
	} else {
		auth = []ssh.AuthMethod{ssh.Password(password)}
	}

	// Set up SSH client configuration for the jump host.
	return &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

func Ssh(target *Target, command string) (*string, error) {
	var wg sync.WaitGroup
	var metrics string
	var sessionErr, commandErr error

	targetConfig := SshConfig(target.User,
		target.Key,
		target.Passphrase,
		target.Password)

	// Establish a connection to the host.
	targetConn, err := ssh.Dial("tcp", target.Addr, targetConfig)
	if err != nil {
		slog.Error("Failed to connect to target host: %v", err)
		return nil, err
	}
	defer targetConn.Close()

	wg.Add(1)
	go func(client *ssh.Client) {
		defer wg.Done()
		var session *ssh.Session
		session, sessionErr = client.NewSession()
		if err != nil {
			return
		}
		defer session.Close()

		// Run the command
		var cmdBuff bytes.Buffer
		session.Stdout = &cmdBuff
		commandErr = session.Run(command)
		metrics = cmdBuff.String()
	}(targetConn)

	wg.Wait()

	if sessionErr != nil {
		slog.Error("Failed to create SSH session to the target host: %v", err)
		return nil, err
	}
	if commandErr != nil {
		slog.Error("Failed to run command: ", err)
		return nil, err
	}

	return &metrics, nil
}

func SshOneJump(target *Target, command string) (*string, error) {
	var wg sync.WaitGroup
	var metrics string
	var sessionErr, commandErr error

	targetConfig := SshConfig(
		target.User,
		target.Key,
		target.Passphrase,
		target.Password)
	jumpConfig := SshConfig(
		target.JumpHosts[0].User,
		target.JumpHosts[0].Key,
		target.JumpHosts[0].Passphrase,
		target.JumpHosts[0].Password)

	// Establish a connection to the jump host.
	jumpConn, err := ssh.Dial("tcp", target.JumpHosts[0].Addr, jumpConfig)
	if err != nil {
		slog.Error("Failed to connect to jump host: %v", err)
		return nil, err
	}
	defer jumpConn.Close()

	// Establish a nested SSH connection to the target host through the jump host.
	targetConn, err := jumpConn.Dial("tcp", target.Addr)
	if err != nil {
		slog.Error("Failed to establish a nested connection to the target host: %v", err)
		return nil, err
	}
	defer targetConn.Close()

	ncc, chans, reqs, err := ssh.NewClientConn(targetConn, target.Addr, targetConfig)
	if err != nil {
		slog.Error("Failed to create SSH client connection to the target host: %v", err)
		return nil, err
	}
	defer ncc.Close()

	targetClient := ssh.NewClient(ncc, chans, reqs)
	defer targetClient.Close()

	wg.Add(1)
	go func(client *ssh.Client) {
		defer wg.Done()
		var session *ssh.Session
		session, sessionErr = client.NewSession()
		if err != nil {
			return
		}
		defer session.Close()

		// Run the command
		var cmdBuff bytes.Buffer
		session.Stdout = &cmdBuff
		commandErr = session.Run(command)
		metrics = cmdBuff.String()
	}(targetClient)

	wg.Wait()

	if sessionErr != nil {
		slog.Error("Failed to create SSH session to the target host: %v", err)
		return nil, err
	}
	if commandErr != nil {
		slog.Error("Failed to run command: ", err)
		return nil, err
	}

	return &metrics, nil
}

func SshTwoJump(target *Target, command string) (*string, error) {
	var wg sync.WaitGroup
	var metrics string
	var sessionErr, commandErr error

	targetConfig := SshConfig(
		target.User,
		target.Key,
		target.Passphrase,
		target.Password)
	jumpConfig1 := SshConfig(
		target.JumpHosts[0].User,
		target.JumpHosts[0].Key,
		target.JumpHosts[0].Passphrase,
		target.JumpHosts[0].Password)
	jumpConfig2 := SshConfig(
		target.JumpHosts[1].User,
		target.JumpHosts[1].Key,
		target.JumpHosts[1].Passphrase,
		target.JumpHosts[1].Password)

	// Establish a connection to the first jump host.
	jumpConn1, err := ssh.Dial("tcp", target.JumpHosts[0].Addr, jumpConfig1)
	if err != nil {
		slog.Error("Failed to connect to jump host 1: %v", err)
		return nil, err
	}
	defer jumpConn1.Close()

	// Establish a connection to the second jump host.
	jumpConn2, err := jumpConn1.Dial("tcp", target.JumpHosts[1].Addr)
	if err != nil {
		slog.Error("Failed to connect to jump host 1: %v", err)
		return nil, err
	}
	defer jumpConn2.Close()
	jumpncc2, chans, reqs, err := ssh.NewClientConn(jumpConn2, target.JumpHosts[1].Addr, jumpConfig2)
	if err != nil {
		slog.Error("Failed to create SSH client connection to the target host: %v", err)
		return nil, err
	}
	defer jumpncc2.Close()
	jumpClient2 := ssh.NewClient(jumpncc2, chans, reqs)
	defer jumpClient2.Close()

	// Establish a nested SSH connection to the target host through the jump host.
	targetConn, err := jumpClient2.Dial("tcp", target.Addr)
	if err != nil {
		slog.Error("Failed to establish a nested connection to the target host: %v", err)
		return nil, err
	}
	defer targetConn.Close()

	ncc, chans, reqs, err := ssh.NewClientConn(targetConn, target.Addr, targetConfig)
	if err != nil {
		slog.Error("Failed to create SSH client connection to the target host: %v", err)
		return nil, err
	}
	defer ncc.Close()

	targetClient := ssh.NewClient(ncc, chans, reqs)
	defer targetClient.Close()

	wg.Add(1)
	go func(client *ssh.Client) {
		defer wg.Done()
		var session *ssh.Session
		session, sessionErr = client.NewSession()
		if err != nil {
			return
		}
		defer session.Close()

		// Run the command
		var cmdBuff bytes.Buffer
		session.Stdout = &cmdBuff
		commandErr = session.Run(command)
		metrics = cmdBuff.String()
	}(targetClient)

	wg.Wait()

	if sessionErr != nil {
		slog.Error("Failed to create SSH session to the target host: %v", err)
		return nil, err
	}
	if commandErr != nil {
		slog.Error("Failed to run command: ", err)
		return nil, err
	}

	return &metrics, nil
}
