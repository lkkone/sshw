package sshw

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/chzyer/readline"
	pkgsftp "github.com/pkg/sftp"
)

// SFTPClient opens an interactive SFTP session for a configured node.
type SFTPClient interface {
	Run() error
}

type defaultSFTPClient struct {
	connector *defaultClient
	input     io.Reader
	output    io.Writer
}

type sftpSession struct {
	client            *pkgsftp.Client
	lineReader        sftpLineReader
	output            io.Writer
	localDir          string
	remoteDir         string
	directoryProgress *directoryTransferProgress
}

type sftpLineReader interface {
	Readline() (string, error)
	Close() error
}

type scannerLineReader struct {
	scanner *bufio.Scanner
	output  io.Writer
}

func (r *scannerLineReader) Readline() (string, error) {
	fmt.Fprint(r.output, "sftp> ")
	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return r.scanner.Text(), nil
}

func (*scannerLineReader) Close() error {
	return nil
}

// NewSFTPClient creates an interactive SFTP client that reuses the same
// authentication and jump-host configuration as the SSH shell client.
func NewSFTPClient(node *Node) SFTPClient {
	return &defaultSFTPClient{
		connector: genSSHConfig(node),
		input:     os.Stdin,
		output:    os.Stdout,
	}
}

func (c *defaultSFTPClient) Run() error {
	sshClient, cleanup, err := c.connector.connect()
	if err != nil {
		return err
	}
	defer cleanup()

	client, err := pkgsftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("start sftp subsystem: %w", err)
	}
	defer client.Close()

	localDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get local working directory: %w", err)
	}
	remoteDir, err := client.Getwd()
	if err != nil {
		remoteDir = "/"
	}

	fmt.Fprintf(
		c.output,
		"Connected to %s@%s using SFTP.\nType \"help\" for available commands.\n",
		c.connector.node.user(),
		c.connector.node.Host,
	)

	lineReader, err := newSFTPLineReader(c.input, c.output)
	if err != nil {
		return fmt.Errorf("initialize sftp command line: %w", err)
	}
	defer lineReader.Close()

	session := &sftpSession{
		client:     client,
		lineReader: lineReader,
		output:     c.output,
		localDir:   localDir,
		remoteDir:  path.Clean(remoteDir),
	}
	return session.run()
}

func (s *sftpSession) run() error {
	for {
		line, err := s.lineReader.Readline()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) {
				continue
			}
			if !errors.Is(err, io.EOF) {
				return fmt.Errorf("read sftp command: %w", err)
			}
			fmt.Fprintln(s.output)
			return nil
		}

		args, err := splitSFTPCommand(line)
		if err != nil {
			fmt.Fprintf(s.output, "error: %v\n", err)
			continue
		}
		if len(args) == 0 {
			continue
		}

		exit, err := s.execute(args)
		if err != nil {
			fmt.Fprintf(s.output, "error: %v\n", err)
		}
		if exit {
			return nil
		}
	}
}

func newSFTPLineReader(input io.Reader, output io.Writer) (sftpLineReader, error) {
	if input != os.Stdin {
		return &scannerLineReader{
			scanner: bufio.NewScanner(input),
			output:  output,
		}, nil
	}

	return readline.NewEx(&readline.Config{
		Prompt:            "sftp> ",
		HistoryLimit:      200,
		HistorySearchFold: true,
		AutoComplete: readline.NewPrefixCompleter(
			readline.PcItem("pwd"),
			readline.PcItem("lpwd"),
			readline.PcItem("ls"),
			readline.PcItem("lls"),
			readline.PcItem("cd"),
			readline.PcItem("lcd"),
			readline.PcItem("get"),
			readline.PcItem("reget"),
			readline.PcItem("put"),
			readline.PcItem("reput"),
			readline.PcItem("mkdir"),
			readline.PcItem("rm"),
			readline.PcItem("rmdir"),
			readline.PcItem("rename"),
			readline.PcItem("help"),
			readline.PcItem("exit"),
			readline.PcItem("quit"),
			readline.PcItem("bye"),
		),
		Stdin:  os.Stdin,
		Stdout: output,
		Stderr: output,
	})
}

func (s *sftpSession) execute(args []string) (bool, error) {
	switch strings.ToLower(args[0]) {
	case "exit", "quit", "bye":
		return true, nil
	case "help", "?":
		s.printHelp()
	case "pwd":
		fmt.Fprintln(s.output, s.remoteDir)
	case "lpwd":
		fmt.Fprintln(s.output, s.localDir)
	case "ls":
		return false, s.listRemote(optionalArg(args))
	case "lls":
		return false, s.listLocal(optionalArg(args))
	case "cd":
		return false, s.changeRemoteDir(requiredArg(args, "cd <remote-path>"))
	case "lcd":
		return false, s.changeLocalDir(requiredArg(args, "lcd <local-path>"))
	case "get":
		options, err := parseTransferArgs(args[1:], true)
		if err != nil {
			return false, fmt.Errorf("usage: get [-f] [-r] <remote-path> [local-path]")
		}
		if options.Recursive {
			return false, s.downloadRecursive(
				options.Paths[0],
				optionalPath(options.Paths),
				options.Force,
			)
		}
		return false, s.download(
			options.Paths[0],
			optionalPath(options.Paths),
			options.Force,
		)
	case "reget":
		options, err := parseTransferArgs(args[1:], false)
		if err != nil {
			return false, fmt.Errorf("usage: reget [-f] <remote-path> [local-path]")
		}
		return false, s.reget(
			options.Paths[0],
			optionalPath(options.Paths),
			options.Force,
		)
	case "put":
		options, err := parseTransferArgs(args[1:], true)
		if err != nil {
			return false, fmt.Errorf("usage: put [-f] [-r] <local-path> [remote-path]")
		}
		if options.Recursive {
			return false, s.uploadRecursive(
				options.Paths[0],
				optionalPath(options.Paths),
				options.Force,
			)
		}
		return false, s.upload(
			options.Paths[0],
			optionalPath(options.Paths),
			options.Force,
		)
	case "reput":
		options, err := parseTransferArgs(args[1:], false)
		if err != nil {
			return false, fmt.Errorf("usage: reput [-f] <local-path> [remote-path]")
		}
		return false, s.reput(
			options.Paths[0],
			optionalPath(options.Paths),
			options.Force,
		)
	case "mkdir":
		remotePath, err := requiredArg(args, "mkdir <remote-path>")
		if err != nil {
			return false, err
		}
		return false, s.client.Mkdir(s.resolveRemote(remotePath))
	case "rm":
		remotePath, err := requiredArg(args, "rm <remote-path>")
		if err != nil {
			return false, err
		}
		return false, s.client.Remove(s.resolveRemote(remotePath))
	case "rmdir":
		remotePath, err := requiredArg(args, "rmdir <remote-path>")
		if err != nil {
			return false, err
		}
		return false, s.client.RemoveDirectory(s.resolveRemote(remotePath))
	case "rename":
		if len(args) != 3 {
			return false, fmt.Errorf("usage: rename <old-path> <new-path>")
		}
		return false, s.client.Rename(s.resolveRemote(args[1]), s.resolveRemote(args[2]))
	default:
		return false, fmt.Errorf("unknown command %q; type \"help\" for available commands", args[0])
	}
	return false, nil
}

func (s *sftpSession) printHelp() {
	fmt.Fprintln(s.output, `Available commands:
  pwd                          show remote working directory
  lpwd                         show local working directory
  ls [remote-path]             list remote directory
  lls [local-path]             list local directory
  cd <remote-path>             change remote directory
  lcd <local-path>             change local directory
  get [-f] [-r] <remote> [local] download a file or directory
  reget [-f] <remote> [local]    resume an interrupted download
  put [-f] [-r] <local> [remote] upload a file or directory
  reput [-f] <local> [remote]    resume an interrupted upload
  mkdir <remote-path>          create remote directory
  rm <remote-path>             remove remote file
  rmdir <remote-path>          remove empty remote directory
  rename <old> <new>           rename remote path
  help                         show this help
  exit                         close the session`)
}

func (s *sftpSession) listRemote(value string, err error) error {
	if err != nil {
		return err
	}
	target := s.remoteDir
	if value != "" {
		target = s.resolveRemote(value)
	}
	entries, err := s.client.ReadDir(target)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		printFileInfo(s.output, entry)
	}
	return nil
}

func (s *sftpSession) listLocal(value string, err error) error {
	if err != nil {
		return err
	}
	target := s.localDir
	if value != "" {
		target = s.resolveLocal(value)
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return err
		}
		printFileInfo(s.output, info)
	}
	return nil
}

func printFileInfo(output io.Writer, info os.FileInfo) {
	fmt.Fprintf(
		output,
		"%s %12d %s %s\n",
		info.Mode(),
		info.Size(),
		info.ModTime().Format("2006-01-02 15:04"),
		info.Name(),
	)
}

func (s *sftpSession) changeRemoteDir(value string, err error) error {
	if err != nil {
		return err
	}
	target := s.resolveRemote(value)
	info, err := s.client.Stat(target)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", target)
	}
	s.remoteDir = target
	return nil
}

func (s *sftpSession) changeLocalDir(value string, err error) error {
	if err != nil {
		return err
	}
	target := s.resolveLocal(value)
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", target)
	}
	s.localDir = target
	return nil
}

func (s *sftpSession) resolveRemote(value string) string {
	if path.IsAbs(value) {
		return path.Clean(value)
	}
	return path.Clean(path.Join(s.remoteDir, value))
}

func (s *sftpSession) resolveLocal(value string) string {
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(s.localDir, value))
}

func optionalArg(args []string) (string, error) {
	if len(args) > 2 {
		return "", errors.New("too many arguments")
	}
	if len(args) == 1 {
		return "", nil
	}
	return args[1], nil
}

func requiredArg(args []string, usage string) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("usage: %s", usage)
	}
	return args[1], nil
}

func optionalPath(paths []string) string {
	if len(paths) == 2 {
		return paths[1]
	}
	return ""
}

type transferArgs struct {
	Force     bool
	Recursive bool
	Paths     []string
}

func parseTransferArgs(args []string, allowRecursive bool) (transferArgs, error) {
	var options transferArgs
	parseFlags := true
	for _, argument := range args {
		if parseFlags && argument == "--" {
			parseFlags = false
			continue
		}
		if parseFlags && strings.HasPrefix(argument, "-") {
			switch argument {
			case "-f":
				options.Force = true
			case "-r":
				if !allowRecursive {
					return transferArgs{}, errors.New("recursive resume is not supported")
				}
				options.Recursive = true
			default:
				return transferArgs{}, fmt.Errorf("unknown option %s", argument)
			}
			continue
		}
		options.Paths = append(options.Paths, argument)
	}
	if len(options.Paths) < 1 || len(options.Paths) > 2 {
		return transferArgs{}, errors.New("invalid transfer arguments")
	}
	return options, nil
}

func splitSFTPCommand(input string) ([]string, error) {
	var (
		args         []string
		current      strings.Builder
		quote        rune
		escaped      bool
		tokenStarted bool
	)

	flush := func() {
		if tokenStarted {
			args = append(args, current.String())
			current.Reset()
			tokenStarted = false
		}
	}

	for _, char := range input {
		if escaped {
			if !unicode.IsSpace(char) && char != '\\' && char != '\'' && char != '"' {
				current.WriteRune('\\')
			}
			current.WriteRune(char)
			tokenStarted = true
			escaped = false
			continue
		}
		if char == '\\' && quote != '\'' {
			escaped = true
			tokenStarted = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			} else {
				current.WriteRune(char)
			}
			tokenStarted = true
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			tokenStarted = true
			continue
		}
		if unicode.IsSpace(char) {
			flush()
			continue
		}
		current.WriteRune(char)
		tokenStarted = true
	}

	if escaped {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, errors.New("unterminated quoted string")
	}
	flush()
	return args, nil
}
