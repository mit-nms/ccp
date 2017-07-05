// +build linux

package main

import (
    "fmt"

    "ccp/ccpFlow/pattern"
    "ccp/ipc"
    "ccp/netlinkipc"

	log "github.com/sirupsen/logrus"
)

func main() {
    nl, err := netlinkipc.New().SetupSend("", 33).SetupListen("", 33).SetupFinish()
    if err != nil {
        log.WithFields(log.Fields{
            "where": "recv setup",
        }).Warn(err)
        return
    }

    i, _ := ipc.SetupWithBackend(nl)
    testControl(i)
}

func testControl(i *ipc.Ipc) {
    cwnd := uint32(10) * 1460
    crs, _ := i.ListenCreateMsg()
    mrs, _ := i.ListenMeasureMsg()
    for {
        select {
        case cr := <-crs:
            log.WithFields(log.Fields{
                "got": fmt.Sprintf("(%v, %v)", cr.SocketId(), cr.CongAlg()),
            }).Info("rcvd create")
        case ack := <-mrs:
            log.WithFields(log.Fields{
                "got":  fmt.Sprintf("(%v, %v, %v)", ack.SocketId(), ack.AckNo(), ack.Rtt()),
                "cwnd": cwnd,
            }).Info("rcvd ack")
        }

        cwnd += 2 * 1460
        staticPattern, err := pattern.NewPattern().
        Cwnd(cwnd).
        Compile()
        if err != nil {
            log.WithFields(log.Fields{
                "err":  err,
                "cwnd": cwnd,
            }).Info("send cwnd msg failed")
            continue
        }

        i.SendPatternMsg(42, staticPattern)
        log.WithFields(log.Fields{
            "cwnd": cwnd,
        }).Info("sent cwnd msg")
    }
}
