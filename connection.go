package sshw

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

func (c *defaultClient) connect() (*ssh.Client, cleanupFunc, error) {
	var (
		cleanups []cleanupFunc
		once     sync.Once
	)

	cleanups = append(cleanups, c.close)
	cleanup := func() {
		once.Do(func() {
			for i := len(cleanups) - 1; i >= 0; i-- {
				cleanups[i]()
			}
		})
	}
	fail := func(err error) (*ssh.Client, cleanupFunc, error) {
		cleanup()
		return nil, func() {}, err
	}

	host := c.node.Host
	port := strconv.Itoa(c.node.port())
	targetAddress := net.JoinHostPort(host, port)

	if len(c.node.Jump) == 0 {
		client, err := c.dialDirect(targetAddress)
		if err != nil {
			return fail(err)
		}
		cleanups = append(cleanups, func() { _ = client.Close() })
		return client, cleanup, nil
	}

	jumpNode := c.node.Jump[0]
	jumpClientConfig := genSSHConfig(jumpNode)
	cleanups = append(cleanups, jumpClientConfig.close)

	proxyClient, err := ssh.Dial(
		"tcp",
		net.JoinHostPort(jumpNode.Host, strconv.Itoa(jumpNode.port())),
		jumpClientConfig.clientConfig,
	)
	if err != nil {
		return fail(err)
	}
	cleanups = append(cleanups, func() { _ = proxyClient.Close() })

	conn, err := proxyClient.Dial("tcp", targetAddress)
	if err != nil {
		return fail(err)
	}
	cleanups = append(cleanups, func() { _ = conn.Close() })

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, targetAddress, c.clientConfig)
	if err != nil {
		return fail(err)
	}
	client := ssh.NewClient(clientConn, chans, reqs)
	cleanups = append(cleanups, func() { _ = client.Close() })

	return client, cleanup, nil
}

func (c *defaultClient) dialDirect(address string) (*ssh.Client, error) {
	client, err := ssh.Dial("tcp", address, c.clientConfig)
	if err == nil {
		return client, nil
	}

	message := err.Error()
	if !strings.Contains(message, "no supported methods remain") || strings.Contains(message, "password") {
		return nil, err
	}

	fmt.Printf("%s@%s's password:", c.clientConfig.User, c.node.Host)
	password, readErr := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if readErr != nil {
		return nil, readErr
	}
	if len(password) == 0 {
		return nil, err
	}

	c.clientConfig.Auth = append(c.clientConfig.Auth, ssh.Password(string(password)))
	return ssh.Dial("tcp", address, c.clientConfig)
}
