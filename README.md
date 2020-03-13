# Bonzomatic Server

This application relays shaders from opponents during LiveShading battle that use [Bonzomatic](https://github.com/GargaJ/Bonzomatic).

Each LiveShading session can happen in different rooms, with multiple opponents.

This application is uses the [Gorilla websocket](https://github.com/gorilla/websocket) package and is based on their chat example.

## How it works

We distinguish:
- Opponents: That use Bonzomatic connected to `ws://server:port/{room_id}/{nickname}` and send their shader progress.
- Viewers: That use Bonzomatic connected to `ws://server:port/{room_id}/{nickname}` and receive shader progress from the `nickname` opponent.

The `room_id` is the secret that can be shared to isolate a LiveShading battle from another.

## Build

### 1- Direct build (if you have go installed)
You need working Go development environment.

Once you have Go up and running, you can download, build and run the example
using the following commands.

    $ git clone https://github.com/alkama/BonzomaticServer.git
    $ cd BonzomaticServer
    $ go build

You'll have a `BonzomaticServer(.exe)` binary produced in the current folder.

Launch it, and you should be able to connect with a websocket client to ws://localhost:9000/testroom/johndoe and send messages.

### 2- Using Docker

#### 2.1- Build:

    $ git clone https://github.com/alkama/BonzomaticServer.git
    $ cd BonzomaticServer
    $ docker build --force-rm --no-cache --rm -t BonzomaticServer .

#### 2.2- Run:

    $ docker run -p 9000:9000 --name BonzomaticServer BonzomaticServer

### 3- Using Docker-Compose

Build and run with the following commands:

    $ git clone https://github.com/alkama/BonzomaticServer.git
    $ cd BonzomaticServer
    $ docker-compose up -d
