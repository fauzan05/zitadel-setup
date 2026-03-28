package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	clientID := flag.String("client-id", "", "OAuth client_id")
	clientSecret := flag.String("client-secret", "", "OAuth client_secret")
	clientIDAlt := flag.String("id", "", "Alias untuk client_id")
	clientSecretAlt := flag.String("secret", "", "Alias untuk client_secret")
	flag.Parse()

	id := strings.TrimSpace(*clientID)
	secret := strings.TrimSpace(*clientSecret)

	if id == "" {
		id = strings.TrimSpace(*clientIDAlt)
	}
	if secret == "" {
		secret = strings.TrimSpace(*clientSecretAlt)
	}

	args := flag.Args()
	if id == "" && len(args) > 0 {
		id = strings.TrimSpace(args[0])
	}
	if secret == "" && len(args) > 1 {
		secret = strings.TrimSpace(args[1])
	}

	reader := bufio.NewReader(os.Stdin)

	if id == "" {
		fmt.Print("Masukkan client_id: ")
		input, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Fprintln(os.Stderr, "Gagal membaca client_id:", err)
			os.Exit(1)
		}
		id = strings.TrimSpace(input)
	}

	if secret == "" {
		fmt.Print("Masukkan client_secret: ")
		input, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Fprintln(os.Stderr, "Gagal membaca client_secret:", err)
			os.Exit(1)
		}
		secret = strings.TrimSpace(input)
	}

	if id == "" || secret == "" {
		fmt.Fprintln(os.Stderr, "client_id dan client_secret wajib diisi")
		os.Exit(1)
	}

	combined := id + ":" + secret
	encoded := base64.StdEncoding.EncodeToString([]byte(combined))

	fmt.Println(encoded)
}
