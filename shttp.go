package shttp

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"io"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type Identity []byte

func NewIdentity() (Identity, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	return privateKeyBytes, nil
}

func (i Identity) Signer() ssh.Signer {
	privateKey, err := x509.ParsePKCS8PrivateKey(i)
	if err != nil {
		panic(err)
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		panic(err)
	}

	return signer
}

const sshChannelName = "http"

type Connection struct {
	w io.ReadWriteCloser
}

func (c *Connection) Read(p []byte) (n int, err error) {
	return c.w.Read(p)
}

func (c *Connection) Write(p []byte) (n int, err error) {
	return c.w.Write(p)
}

func (c *Connection) Close() error {
	return c.w.Close()
}

type ListenOptions struct {
	Address  string
	Identity ssh.Signer
}

type Listener struct {
	options   ListenOptions
	sshConfig *ssh.ServerConfig
	handle    func(conn *Connection)
	l         net.Listener
}

func SetupListener(options ListenOptions, handle func(conn *Connection)) (*Listener, error) {
	sshConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			return &ssh.Permissions{
				Extensions: map[string]string{
					"pubkey-fp": ssh.FingerprintSHA256(pubKey),
				},
			}, nil
		},
	}
	sshConfig.AddHostKey(options.Identity)

	l, err := net.Listen("tcp", options.Address)
	if err != nil {
		return nil, err
	}
	return &Listener{
		options:   options,
		sshConfig: sshConfig,
		handle:    handle,
		l:         l,
	}, nil
}

func (l *Listener) Accept() {
	for {
		c, err := l.l.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			continue
		}
		go l.accept(c)
	}
}

func (l *Listener) Close() {
	l.l.Close()
}

func (l *Listener) accept(c net.Conn) {
	_, chans, reqs, err := ssh.NewServerConn(c, l.sshConfig)
	if err != nil {
		c.Close()
		return
	}

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != sshChannelName {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			return
		}
		channel, _, err := newChannel.Accept()
		if err != nil {
			break
		}
		l.handle(&Connection{channel})
		channel.Close()
	}
}

type DialOptions struct {
	Network  string
	Address  string
	Identity ssh.Signer
	Timeout  time.Duration
}

func Dial(options DialOptions) (*Connection, error) {
	clientConfig := &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(options.Identity),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		HostKeyAlgorithms: []string{ssh.KeyAlgoED25519},
		Timeout:           options.Timeout,
	}

	client, err := ssh.Dial(options.Network, options.Address, clientConfig)
	if err != nil {
		return nil, err
	}

	channel, _, err := client.OpenChannel(sshChannelName, nil)
	if err != nil {
		return nil, err
	}

	return &Connection{channel}, nil
}
