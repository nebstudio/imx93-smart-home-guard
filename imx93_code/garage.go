package main

import (
	"log"

	"imx93-guard/applink"
)

type garageController interface {
	GarageOpen() bool
	SetGarage(open bool) error
}

const (

	garageCarNearCm = 25

	garageCarFarCm = 40

	garageFarConfirmReadings = 5
)

type garageAutoCloser struct {

	sawCarNear bool

	farStreak int

	wasOpen bool
}

func newGarageAutoCloser() *garageAutoCloser {
	return &garageAutoCloser{}
}

func (g *garageAutoCloser) tick(dist DistanceReading, act garageController, link *applink.Server) {
	open := act.GarageOpen()

	if open != g.wasOpen {
		g.sawCarNear = false
		g.farStreak = 0
		g.wasOpen = open
	}

	if !open {
		return
	}

	near := dist.Valid && dist.Cm <= garageCarNearCm
	far := !dist.Valid || dist.Cm > garageCarFarCm

	if near {

		g.sawCarNear = true
		g.farStreak = 0
		return
	}

	if far {

		if !g.sawCarNear {
			return
		}
		g.farStreak++
		if g.farStreak >= garageFarConfirmReadings {
			log.Printf("车库门自动关闭：检测到车已驶离(距离=%dcm 有效=%v)，连续%d帧确认后关门",
				dist.Cm, dist.Valid, g.farStreak)
			if err := act.SetGarage(false); err != nil {
				log.Printf("车库门自动关闭执行失败: %v", err)
				return
			}

			g.sawCarNear = false
			g.farStreak = 0
			g.wasOpen = false
			if link != nil {
				link.BroadcastEvent("garage_auto_closed", "检测到车辆驶离，车库门已自动关闭")
			}
		}
		return
	}

}
