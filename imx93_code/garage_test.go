package main

import "testing"

type fakeGarage struct {
	open      bool
	setCalls  []bool
	setGarage func(open bool) error
}

func (f *fakeGarage) GarageOpen() bool { return f.open }

func (f *fakeGarage) SetGarage(open bool) error {
	f.setCalls = append(f.setCalls, open)
	if f.setGarage != nil {
		return f.setGarage(open)
	}
	f.open = open
	return nil
}

func TestGarageAutoClose_NearThenFar_Closes(t *testing.T) {
	fg := &fakeGarage{open: true}
	g := newGarageAutoCloser()

	g.tick(DistanceReading{Cm: 20, Valid: true}, fg, nil)
	if !fg.open {
		t.Fatal("车停在近处时不应关门")
	}

	for i := 0; i < garageFarConfirmReadings-1; i++ {
		g.tick(DistanceReading{Cm: 200, Valid: true}, fg, nil)
		if !fg.open {
			t.Fatalf("第%d帧远离(未达确认帧数)不应关门", i+1)
		}
	}
	g.tick(DistanceReading{Cm: 200, Valid: true}, fg, nil)
	if fg.open {
		t.Fatal("车连续多帧远离后应自动关门")
	}

	for i, v := range fg.setCalls {
		if v {
			t.Fatalf("第%d次 SetGarage 调用为true(自动开门)，违反“只关不开”安全原则", i+1)
		}
	}
}

func TestGarageAutoClose_OpenWithoutCar_DoesNotClose(t *testing.T) {
	fg := &fakeGarage{open: true}
	g := newGarageAutoCloser()

	for i := 0; i < garageFarConfirmReadings*3; i++ {
		g.tick(DistanceReading{Cm: 200, Valid: true}, fg, nil)
	}
	if !fg.open {
		t.Fatal("从未检测到车停靠时不应自动关门")
	}
	if len(fg.setCalls) != 0 {
		t.Fatalf("不应有任何关门动作，实际调用了%d次", len(fg.setCalls))
	}
}

func TestGarageAutoClose_NoEchoCountsAsFar(t *testing.T) {
	fg := &fakeGarage{open: true}
	g := newGarageAutoCloser()

	g.tick(DistanceReading{Cm: 20, Valid: true}, fg, nil)

	for i := 0; i < garageFarConfirmReadings; i++ {
		g.tick(DistanceReading{Valid: false}, fg, nil)
	}
	if fg.open {
		t.Fatal("无回波连续多帧应判定车已驶离并自动关门")
	}
}

func TestGarageAutoClose_GarageClosed_NoAction(t *testing.T) {
	fg := &fakeGarage{open: false}
	g := newGarageAutoCloser()
	for i := 0; i < garageFarConfirmReadings*2; i++ {
		g.tick(DistanceReading{Cm: 200, Valid: true}, fg, nil)
	}
	if len(fg.setCalls) != 0 {
		t.Fatalf("车库门本来就关着，不应有任何动作，实际调用了%d次", len(fg.setCalls))
	}
}

func TestGarageAutoClose_HysteresisBandDoesNotClose(t *testing.T) {
	fg := &fakeGarage{open: true}
	g := newGarageAutoCloser()

	g.tick(DistanceReading{Cm: 20, Valid: true}, fg, nil)

	for i := 0; i < garageFarConfirmReadings*2; i++ {
		g.tick(DistanceReading{Cm: 32, Valid: true}, fg, nil)
	}
	if !fg.open {
		t.Fatal("处于迟滞缓冲带内不应关门")
	}
}

func TestGarageAutoClose_ReopenResetsState(t *testing.T) {
	fg := &fakeGarage{open: true}
	g := newGarageAutoCloser()

	g.tick(DistanceReading{Cm: 20, Valid: true}, fg, nil)
	for i := 0; i < garageFarConfirmReadings; i++ {
		g.tick(DistanceReading{Cm: 200, Valid: true}, fg, nil)
	}
	if fg.open {
		t.Fatal("第一轮应已自动关门")
	}

	fg.open = true
	for i := 0; i < garageFarConfirmReadings*2; i++ {
		g.tick(DistanceReading{Cm: 200, Valid: true}, fg, nil)
	}
	if !fg.open {
		t.Fatal("重新开门后未检测到车停靠，不应立即自动关门")
	}
}
