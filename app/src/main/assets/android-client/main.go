package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	setupGlobalResolver()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case s := <-sig:
			log.Printf("[КЛИЕНТ] Сигнал %v, завершаю...", s)
			cancel()
		case <-ctx.Done():
			return
		}
		select {
		case s := <-sig:
			log.Printf("[КЛИЕНТ] Повторный %v, принудительный выход", s)
			os.Exit(1)
		case <-ctx.Done():
		}
	}()

	var pauseFlag int32

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.Contains(line, "error:tunnel stopped") {
				log.Printf("[STDIN] %s", line)
			}
			switch {
			case line == "PAUSE":
				atomic.StoreInt32(&pauseFlag, 1)
			case line == "RESUME":
				atomic.StoreInt32(&pauseFlag, 0)
			case line == "STOP":
				cancel()
				return
			}
		}
	}()

	ppid := os.Getppid()
	go func() {
		for {
			time.Sleep(2 * time.Second)
			if os.Getppid() != ppid {
				os.Exit(0)
			}
		}
	}()

	host := flag.String("turn", "", "переопределить IP TURN")
	port := flag.String("port", "", "переопределить порт TURN")
	listen := flag.String("listen", "127.0.0.1:9000", "локальный адрес")
	callLink := flag.String("call", "", "ссылка на Max звонок")
	peerAddr := flag.String("peer", "", "адрес:порт VPS сервера")
	numW := flag.Int("n", 24, "количество воркеров (кратно 12)")
	deviceID := flag.String("device-id", "unknown", "уникальный ID устройства")
	connPassword := flag.String("password", "", "пароль подключения")
	fingerprint := flag.String("fingerprint", "chrome", "браузерный фингерпринт (chrome, safari, ios, android, firefox)")
	obfsMode := flag.String("obfs", "audio", "режим обфускации (audio/video/datachannel)")

	flag.Parse()

	if *peerAddr == "" || *callLink == "" {
		log.Fatal("[КЛИЕНТ] Нужны -peer и -call")
	}

	cleanPeerAddr := strings.TrimSpace(*peerAddr)
	var err error
	var peer *net.UDPAddr
	for i := 0; i < 15; i++ {
		peer, err = net.ResolveUDPAddr("udp", cleanPeerAddr)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("[КЛИЕНТ] Ошибка разбора пира: %v", err)
	}

	if *fingerprint != "" {
		SetActiveFingerprint(*fingerprint)
	}

	hashes := ParseHashes(*callLink)
	if len(hashes) == 0 {
		log.Fatal("[КЛИЕНТ] Нет хешей звонка")
	}

	if *connPassword == "" {
		log.Fatal("[КЛИЕНТ] Нужен -password: WRAP ключ выводится из пароля подключения")
	}

	wrapKey, err := deriveWrapKey(*connPassword)
	if err != nil {
		log.Fatalf("[КЛИЕНТ] WRAP key derive: %v", err)
	}

	maxWorkers := 108
	if *numW > maxWorkers {
		*numW = maxWorkers
	}
	if *numW < workersPerGroup {
		*numW = workersPerGroup
	}
	*numW = (*numW / workersPerGroup) * workersPerGroup

	tp := &TurnParams{
		Host:     *host,
		Port:     *port,
		Hashes:   hashes,
		WrapKey:  wrapKey,
		ObfsMode: *obfsMode,
	}

	var localConn net.PacketConn
	actualListenAddr := *listen
	for i := 0; i < 5; i++ {
		localConn, err = net.ListenPacket("udp", actualListenAddr)
		if err == nil {
			break
		}
		log.Printf("[ОЖИДАНИЕ] Порт %s занят (возможно, старый процесс завершается). Жду... (%d/5)", actualListenAddr, i+1)
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		log.Printf("[АВТО-ПОРТ] Порт %s всё ещё занят. Пробую случайный динамический порт...", actualListenAddr)
		actualListenAddr = "127.0.0.1:0"
		localConn, err = net.ListenPacket("udp", actualListenAddr)
		if err != nil {
			log.Fatalf("[ФАТАЛ] Ошибка бинда динамического порта: %v", err)
		}
	}
	if uc, ok := localConn.(*net.UDPConn); ok {
		_ = uc.SetReadBuffer(socketBufSize)
		_ = uc.SetWriteBuffer(socketBufSize)
	}
	stopLocalConn := context.AfterFunc(ctx, func() { _ = localConn.Close() })
	defer stopLocalConn()

	_, localPort, _ := net.SplitHostPort(localConn.LocalAddr().String())
	if localPort == "" {
		localPort = "9000"
	}

	numGroups := *numW / workersPerGroup

	wrapStatus := "OFF"
	if len(wrapKey) == wrapKeyLen {
		wrapStatus = "ON (password HKDF + RTP AEAD)"
	}

	log.Println("[КЛИЕНТ] ═══════════════════════════════════════")
	log.Printf("[КЛИЕНТ] Provider: Max (call=%s)", hashes[0])
	log.Printf("[КЛИЕНТ] TLS: %s fingerprint", GetActiveFingerprint())
	log.Printf("[КЛИЕНТ] Воркеров: %d (групп: %d, по %d)", *numW, numGroups, workersPerGroup)
	log.Printf("[КЛИЕНТ] Хешей: %d", len(hashes))
	log.Printf("[КЛИЕНТ] Слушаю: %s | Пир: %s", *listen, cleanPeerAddr)
	log.Printf("[КЛИЕНТ] WRAP: %s", wrapStatus)
	log.Printf("[WRAP] Ключ выведен из пароля, режим RTP AEAD активен")
	log.Printf("[КЛИЕНТ] Device ID: %s", *deviceID)
	log.Println("[КЛИЕНТ] ═══════════════════════════════════════")

	stats := NewStats()
	shutdownCh := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(shutdownCh)
	}()
	go stats.RunLoop(shutdownCh)

	disp := NewDispatcher(ctx, localConn, stats)
	defer disp.Shutdown()

	configCh := make(chan string, 1)
	configDone := make(chan struct{})
	go func() {
		defer close(configDone)
		select {
		case rawConf, ok := <-configCh:
			if !ok || rawConf == "" {
				return
			}
			finalConf := rawConf
			if !strings.Contains(finalConf, "MTU =") {
				lines := strings.Split(finalConf, "\n")
				var newLines []string
				for _, line := range lines {
					newLines = append(newLines, line)
					if strings.TrimSpace(line) == "[Interface]" {
						newLines = append(newLines, "MTU = 1280")
					}
				}
				finalConf = strings.Join(newLines, "\n")
			}
			fmt.Println()
			fmt.Println("╔══════════════ WireGuard Конфиг ══════════════╗")
			for _, line := range strings.Split(finalConf, "\n") {
				fmt.Printf("║ %-44s ║\n", line)
			}
			fmt.Println("╚══════════════════════════════════════════════╝")
			if err := os.WriteFile("wg-turn.conf", []byte(finalConf+"\n"), 0600); err != nil {
				log.Printf("[КОНФИГ] Ошибка сохранения: %v", err)
			} else {
				log.Println("[КОНФИГ] Сохранён в wg-turn.conf")
			}
			emitConfig(finalConf)
		case <-ctx.Done():
		}
	}()

	var wg sync.WaitGroup
	workerIDCounter := 1

	var prevWaitCreds <-chan struct{}
	var prevWaitSpawn <-chan struct{}

	for g := 0; g < numGroups; g++ {
		isFirst := (g == 0)

		var myWaitCreds <-chan struct{}
		var mySignalCreds chan<- struct{}
		var myWaitSpawn <-chan struct{}
		var mySignalSpawn chan<- struct{}

		if g > 0 {
			myWaitCreds = prevWaitCreds
			myWaitSpawn = prevWaitSpawn
		}
		if g < numGroups-1 {
			chC := make(chan struct{})
			mySignalCreds = chC
			prevWaitCreds = chC

			chS := make(chan struct{})
			mySignalSpawn = chS
			prevWaitSpawn = chS
		}

		ids := make([]int, workersPerGroup)
		for i := range ids {
			ids[i] = workerIDCounter
			workerIDCounter++
		}

		gID := g + 1
		var cc chan<- string
		if isFirst {
			cc = configCh
		}

		wg.Add(1)
		go func(groupID int, isFirstGroup bool, configChan chan<- string, workerIds []int, startHashIndex int, waitC, waitS <-chan struct{}, sigC, sigS chan<- struct{}) {
			defer wg.Done()
			WorkerGroup(ctx, groupID, startHashIndex, tp, peer, disp, localPort,
				isFirstGroup, configChan, workerIds, &pauseFlag, *deviceID, *connPassword, stats, waitC, sigC, waitS, sigS)
		}(gID, isFirst, cc, ids, g, myWaitCreds, myWaitSpawn, mySignalCreds, mySignalSpawn)
	}

	wg.Wait()
	close(configCh)
	<-configDone
	log.Println("[КЛИЕНТ] Все воркеры завершены")
}
