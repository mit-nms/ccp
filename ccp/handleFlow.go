package main

import (
	"time"

	"ccp/ccpFlow"

	log "github.com/sirupsen/logrus"
)

/* The event loop for a single flow
 * Receive filtered messages from the CCP corresp
 * to this flow, and call the appropriate handling function
 */
func handleFlow(
	sockId uint32,
	flow ccpFlow.Flow,
	msgs flowHandler,
	endFlow chan uint32,
) {
	for {
		select {
		case m := <-msgs.flowMeasureCh:
			log.WithFields(log.Fields{
				"flowid": m.SocketId(),
				"ackno":  m.AckNo(),
				"rtt":    m.Rtt(),
				"rin":    m.Rin(),
				"rout":   m.Rout(),
			}).Debug("handleMeasure")

			flow.GotMeasurement(ccpFlow.Measurement{
				Ack:  m.AckNo(),
				Rtt:  m.Rtt(),
				Rin:  m.Rin(),
				Rout: m.Rout(),
			})
		case dr := <-msgs.flowDropCh:
			log.WithFields(log.Fields{
				"flowid":  dr.SocketId,
				"drEvent": dr.Event,
			}).Debug("handleDrop")
			flow.Drop(ccpFlow.DropEvent(dr.Event()))
		case <-time.After(time.Minute):
			// garbage collect this goroutine after a minute of inactivity
			endFlow <- sockId
			return
		}
	}
}
