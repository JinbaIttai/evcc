package charger

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"

	//"github.com/davecgh/go-spew/spew"
)


// TeslaAPI is an api.Vehicle implementation for Tesla cars
type TeslaAPI struct {
	log     *util.Logger
	lp      loadpoint.API
	enabled bool
}

func init() {
	registry.Add("teslaapi", NewTeslaAPIFromConfig)
}

// NewTeslaAPIFromConfig creates a new vehicle
func NewTeslaAPIFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		VIN    string
		Cache  time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &TeslaAPI{
	}
	//v.lp.SetTargetSoc(cc.TargetSoc)
        //fmt.Println("res is", res);
	//v, ok := v.lp.publishChargerFeature(api.IntegratedDevice)
	//if !ok {
        //       fmt.Println("Soc failed", soc);
//	}

	//log := util.NewLogger("teslaapi")

        //fmt.Println("Title", v.Title_);
        //fmt.Println("Display", v.vehicle.DisplayName);
	//if v.Title_ == "" {
	//	v.Title_ = v.vehicle.DisplayName
	//}
	//spew.Dump(result.Response.ChargeState.ChargerPower)

	return v, nil;
}

// Enabled implements the api.Charger interface
func (c *TeslaAPI) Enabled() (bool, error) {
	enabled, err := verifyEnabled(c, c.enabled)
	if err == nil {
		c.enabled = enabled
	}
	if errors.Is(err, api.ErrAsleep) {
		err = nil
	}
        fmt.Println("error:", err);

	return enabled, err
}

// apiError converts HTTP 408 error to ErrTimeout
func (v *TeslaAPI) apiError(err error) error {
	if err != nil && err.Error() == "408 Request Timeout" {
		err = api.ErrAsleep
	}
	return err
}

// Enable implements the api.Charger interface
func (c *TeslaAPI) Enable(enable bool) error {
        fmt.Println("Enable")
	if c.lp == nil {
		return errors.New("loadpoint not initialized")
	}

	v, ok := c.lp.GetVehicle().(api.VehicleChargeController)
	if !ok {
		return errors.New("vehicle not capable of start/stop")
	}

	var err error
	if enable {
		err = v.StartCharge()
	} else {
		err = v.StopCharge()
	}

	if err == nil {
		c.enabled = enable
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *TeslaAPI) MaxCurrent(current int64) error {
	if c.lp == nil {
		return errors.New("loadpoint not initialized")
	}

	v, ok := c.lp.GetVehicle().(api.CurrentLimiter)
	if !ok {
		return errors.New("vehicle not capable of current control")
	}

	return v.MaxCurrent(current)
}

// Status implements the api.ChargeState interface
func (c *TeslaAPI) Status() (api.ChargeStatus, error) {
	if c.lp == nil {
		return api.StatusA, errors.New("loadpoint not initialized")
	}

	v, ok := c.lp.GetVehicle().(api.ChargeState)
	if !ok {
		return api.StatusA, errors.New("vehicle has no status")
	}

	return v.Status()
}

var _ api.ChargeRater = (*TeslaAPI)(nil)

func (c *TeslaAPI) ChargedEnergy() (float64, error) {
	if c.lp == nil {
		return 0, errors.New("loadpoint not initialized")
	}

	v, ok := c.lp.GetVehicle().(api.ChargeRater)
	if !ok {
		return 0, errors.New("vehicle has no status")
	}

	return v.ChargedEnergy()
}

var _ api.PhaseCurrents = (*TeslaAPI)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *TeslaAPI) Currents() (float64, float64, float64, error) {
	if c.lp == nil {
		return 0, 0, 0, errors.New("loadpoint not initialized")
	}

	v, ok := c.lp.GetVehicle().(api.PhaseCurrents)
	if !ok {
		return 0, 0, 0, errors.New("Currents: vehicle unavailable")
	}

	return v.Currents()
}

var _ api.Meter = (*TeslaAPI)(nil)

// CurrentPower implements the api.Meter interface
func (c *TeslaAPI) CurrentPower() (float64, error) {
	if c.lp == nil {
		return 0, errors.New("loadpoint not initialized")
	}

	v, ok := c.lp.GetVehicle().(api.Meter)
	if !ok {
		return 0, errors.New("Power: vehicle unavailable")
	}

	return v.CurrentPower()
}


var _ api.PhaseVoltages = (*TeslaAPI)(nil)

// Voltages implements the api.PhaseVoltages interface
func (c *TeslaAPI) Voltages() (float64, float64, float64, error) {
	if c.lp == nil {
		return 0, 0, 0, errors.New("loadpoint not initialized")
	}

	v, ok := c.lp.GetVehicle().(api.PhaseVoltages)
	if !ok {
		return 0, 0, 0, errors.New("Power: vehicle unavailable")
	}

	return v.Voltages()
}

var _ loadpoint.Controller = (*TeslaAPI)(nil)

// LoadpointControl implements loadpoint.Controller
func (v *TeslaAPI) LoadpointControl(lp loadpoint.API) {
	v.lp = lp
}

