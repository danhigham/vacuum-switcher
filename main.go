package main

import (
	"log"
	"time"
	"os"
	"os/signal"
	"fmt"
	"reflect"
	"github.com/simulatedsimian/joystick"
	"github.com/tarm/serial"
)

const VacSegmentLength = 178
const HomeCmd = "G10 L20 P1 Z0\n"
const MoveToPosCmd = "G90 G00 Z-%d\n"
const JogLeftCmd = "G91 G00 Z5\n"
const JogRightCmd= "G91 G00 Z-5\n"

type JoystickEvent struct {
	Axis []int
	Buttons []int
}

func sum(array []int) int {
	result := 0
	for _, v := range array {
		result += v
	}
	return result
}

func sendGRBL(s *serial.Port, msg string) (error, string) {

	log.Printf("Sending : %s", msg)
	n, err := s.Write([]byte(msg))
	if err != nil {
		return err, ""
	}

	buf := make([]byte, 128)
	n = -1
	out := ""
	for n != 0 {
		n, _ = s.Read(buf)
		out = fmt.Sprintf("%s%s", out, buf[:n])
	}
	return err, out
}

func main() {
	c := &serial.Config{Name: "/dev/ttyACM0", Baud: 115200, ReadTimeout: time.Millisecond * 500}

	s, err := serial.OpenPort(c)
	if err != nil {
		panic(err)
	}
	defer s.Close()

	err, rsp := sendGRBL(s, "?")
	log.Printf("%s\n", rsp)

	jsid := 0
	js, jserr := joystick.Open(jsid)
	if jserr != nil {
		panic(jserr)
	}

	events := make(chan JoystickEvent)
	go func() {
		lastEvent := JoystickEvent{}
		for true {
			je := JoystickEvent{}

			jinfo, err := js.Read()
			if err != nil {
				panic(err)
			}

			for button := 0; button < js.ButtonCount(); button++ {
				if jinfo.Buttons&(1<<uint32(button)) != 0 {
					je.Buttons = append(je.Buttons, button)
				}
			}

			for axis := 0; axis < js.AxisCount(); axis++ {
				je.Axis = append(je.Axis, jinfo.AxisData[axis])
			}

			if (!reflect.DeepEqual(je, lastEvent)) {
				if sum(je.Axis) != 0 || len(je.Buttons) > 0 {
					events <- je
				}
				lastEvent = je
			}
		}
	}()

	escChan := make(chan os.Signal, 1)
	signal.Notify(escChan, os.Interrupt)

	go func() {
		for range escChan {
			log.Println("Closing connection")
			s.Close()
			os.Exit(0)
		}
	}()


	for {
		ev := <-events
		rsp := ""

		if len(ev.Buttons) == 1 {
			if ev.Buttons[0] < 4 {
				err, rsp = sendGRBL(s, fmt.Sprintf(MoveToPosCmd, ev.Buttons[0] * VacSegmentLength))
				if err == nil {
					log.Printf(">>> %s", rsp)
				}
			}
			// move to a particular segment
			continue
		}

		if reflect.DeepEqual(ev.Buttons, []int{4,5}) {
			// set home
			err, rsp = sendGRBL(s, HomeCmd)
			if err == nil {
				log.Printf(">>> %s", rsp)
			}
			continue
		}

		if ev.Axis[0] != 0 {
			if ev.Axis[0] > 0 {
				err, rsp = sendGRBL(s, JogRightCmd)
			} else {
				err, rsp = sendGRBL(s, JogLeftCmd)
			}
			if err == nil {
				log.Printf(">>> %s", rsp)
			}
			// jog one way or another
		}

	}
}
