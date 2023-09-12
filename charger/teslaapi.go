package charger

import (
	"context"
	"errors"
	"fmt"
	//"math"
	//"strings"
	"slices"
	"time"

	"github.com/bogosj/tesla"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"

	//"github.com/davecgh/go-spew/spew"
)

const (
        interval = 15 * time.Minute // refresh interval when charging
)


// TeslaAPI is an api.Vehicle implementation for Tesla cars
type TeslaAPI struct {
	*embed
	vehicle *tesla.Vehicle
	log     *util.Logger
	lp      loadpoint.API
	dataG   func() (*tesla.VehicleData, error)
	enabled bool
}

func init() {
	registry.Add("teslaapi", NewTeslaAPIFromConfig)
}

// NewTeslaAPIFromConfig creates a new vehicle
func NewTeslaAPIFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		embed  `mapstructure:",squash"`
		AccessToken    string
		RefreshToken    string
		TargetSoc	int
		VIN    string
		Cache  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &TeslaAPI{
		embed: &cc.embed,
	}
	//v.lp.SetTargetSoc(cc.TargetSoc)
        //fmt.Println("res is", res);
	//v, ok := v.lp.publishChargerFeature(api.IntegratedDevice)
	//if !ok {
        //       fmt.Println("Soc failed", soc);
//	}

	// authenticated http client with logging injected to the Tesla client
	log := util.NewLogger("teslaapi").Redact(cc.AccessToken, cc.RefreshToken)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewClient(log))

	options := []tesla.ClientOption{tesla.WithToken(&oauth2.Token{
		AccessToken:  cc.AccessToken,
		RefreshToken: cc.RefreshToken,
		Expiry:       time.Now(),
	})}

	client, err := tesla.NewClient(ctx, options...)
	if err != nil {
		return nil, err
	}
	v.vehicle, err = ensureChargerEx(
		cc.VIN, client.Vehicles,
		func(v *tesla.Vehicle) string {
			return v.Vin
		},
	)

	if err != nil {
		return nil, err
	}

        //fmt.Println("Title", v.Title_);
        //fmt.Println("Display", v.vehicle.DisplayName);
	//if v.Title_ == "" {
	//	v.Title_ = v.vehicle.DisplayName
	//}
        v.dataG = provider.Cached(func() (*tesla.VehicleData, error) {
		res, err := v.vehicle.Data()
		return res, v.apiError(err)
        }, cc.Cache)
        //v.dataG = func() (*tesla.VehicleData, error) {
	//	res, err := v.vehicle.Data()
	//	return res, v.apiError(err)
        //}
        //log.DEBUG.Println("power", v.dataG.Response.ChargeState.ChargerPower);
        //v.dataG, err = v.vehicle.Data()
        //log.DEBUG.Println(v.vehicle.Data())

        //log.DEBUG.Println(v.vehicle.Data())
        // result, err := v.vehicle.Data()
	//spew.Dump(result.Response.ChargeState.ChargerPower)

        //log.DEBUG.Println(v.dataG.Response)
        //log.DEBUG.Println("state", v.vehicle.State);
        //log.DEBUG.Println("foo");
	//v.log.DEBUG.Println("power ", result.Response.ChargeState.ChargerPower)
	//spew.Dump(result.Response.ChargeState.ChargerPower)
        //fmt.Println("foobar");
        //x := int(result.Response.ChargeState.ChargerPower)
        //fmt.Println("foobarbaz");
	//fmt.Println(x)

        //fmt.Println("bar");

	return v, nil;
}

// Enabled implements the api.Charger interface
func (c *TeslaAPI) Enabled() (bool, error) {
        //soc := c.TargetSoc
        //fmt.Println("Soc is", soc);
	//t := c.lp.Title
	//t, ok := c.lp.GetVehicle()
	//t := c.lp.SetVehicle("sonic")
	//if !ok {
	//	fmt.Println("oops")
	//}
        //fmt.Println("t is", t);
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
func (v *TeslaAPI) Enable(enable bool) error {
        fmt.Println("Enable")
	//v, ok := c.lp.GetVehicle().(api.VehicleChargeController)
	//if !ok {
	//	return errors.New("vehicle not capable of start/stop")
	//}

	var err error
	if enable {
		fmt.Println("starting charge")
		err := v.apiError(v.vehicle.StartCharging())
		if err != nil && slices.Contains([]string{"complete", "is_charging"}, err.Error()) {
			return nil
		}
	return err

	} else {
		fmt.Println("stopping charge")
		err := v.apiError(v.vehicle.StopCharging())

		// ignore sleeping vehicle
		if errors.Is(err, api.ErrAsleep) {
			err = nil
		}

	return err
	}

	if err == nil {
		v.enabled = enable
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (v *TeslaAPI) MaxCurrent(current int64) error {
	return v.apiError(v.vehicle.SetChargingAmps(int(current)))
}

// Status implements the api.ChargeState interface
func (v *TeslaAPI) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected
	res, err := v.dataG()
	if err != nil {
	        // ignore sleeping vehicle
	        if errors.Is(err, api.ErrAsleep) {
		      err = nil
               }
		return status, err
	}

	switch res.Response.ChargeState.ChargingState {
	case "Stopped", "NoPower", "Complete":
		status = api.StatusB
	case "Charging":
		status = api.StatusC
	}

	return status, nil
}

var _ api.ChargeRater = (*TeslaAPI)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (v *TeslaAPI) ChargedEnergy() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, nil
	}
	return res.Response.ChargeState.ChargeEnergyAdded, nil
}

// FIXME MDK
//var _ api.ChargeTimer = (*TeslaAPI)(nil) 

//// ChargingTime implements the api.ChargeTimer interface
//func (v *TeslaAPI) ChargingTime() (time.Duration, error) {
//	res, err := v.vitalsG()
//	return time.Duration(res.SessionS) * time.Second, err
//}

// Use workaround if voltageC_v is approximately half of grid_v
//
//	"voltageA_v": 241.5,
//	"voltageB_v": 241.5,
//	"voltageC_v": 118.7,
//
// Default state is ~2V on all phases unless charging

// FIXME MDK
//func (v *TeslaAPI) isSplitPhase(res Vitals) bool {
//	return math.Abs(res.VoltageCV-res.GridV/2) < 25
//}

var _ api.PhaseCurrents = (*TeslaAPI)(nil)

// Currents implements the api.PhaseCurrents interface
func (v *TeslaAPI) Currents() (float64, float64, float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, 0, 0, nil
	}
        phases := res.Response.ChargeState.ChargerPhases;
        current := float64(res.Response.ChargeState.ChargerActualCurrent)
        if phases == 1 {
		return current, 0, 0, err
	} else {
		return current, current, current, err
	}
}

var _ api.Meter = (*TeslaAPI)(nil)

// CurrentPower implements the api.Meter interface
func (v *TeslaAPI) CurrentPower() (float64, error) {
	res, err := v.dataG()
        if err != nil {
          fmt.Println("power error ", 0)
	  // ignore sleeping vehicle
	  if errors.Is(err, api.ErrAsleep) {
		err = nil
	  }
          return 0, err
        }
        voltage := res.Response.ChargeState.ChargerVoltage;
        current := res.Response.ChargeState.ChargerActualCurrent;
        phases := res.Response.ChargeState.ChargerPhases;

        power := ( voltage * current * phases)

        fmt.Println("volts",  voltage)
        fmt.Println("amps",   current)
        fmt.Println("phases", phases)
        fmt.Println("power",  power)
        //return 0, nil
	return float64(power), err
}

var _ api.PhaseVoltages = (*TeslaAPI)(nil)

// Voltages implements the api.PhaseVoltages interface
func (v *TeslaAPI) Voltages() (float64, float64, float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, 0, 0, nil
	}
        phases := res.Response.ChargeState.ChargerPhases;
        voltage := float64(res.Response.ChargeState.ChargerVoltage)
        if phases == 1 {
		return voltage, 0, 0, err
	} else {
		return voltage, voltage, voltage, err
	}
}

var _ loadpoint.Controller = (*TeslaAPI)(nil)

// LoadpointControl implements loadpoint.Controller
func (v *TeslaAPI) LoadpointControl(lp loadpoint.API) {
	v.lp = lp
}

// Soc implements the api.Vehicle interface
func (v *TeslaAPI) Soc() (float64, error) {
	res, err := v.dataG()
	if err != nil {
                fmt.Println("soc error")
		return 0, nil
	}
	return float64(res.Response.ChargeState.UsableBatteryLevel), nil
}


const kmPerMile = 1.609344

var _ api.VehicleRange = (*TeslaAPI)(nil)

// Range implements the api.VehicleRange interface
func (v *TeslaAPI) Range() (int64, error) {
	res, err := v.dataG()
	if err != nil {
                fmt.Println("range error")
		return 0, nil
	}
	// miles to km
	return int64(kmPerMile * res.Response.ChargeState.IdealBatteryRange), nil
}

var _ api.VehicleOdometer = (*TeslaAPI)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *TeslaAPI) Odometer() (float64, error) {
	res, err := v.dataG()
	if err != nil {
                fmt.Println("odo error")
		return 0, nil
	}
	// miles to km
	return kmPerMile * res.Response.VehicleState.Odometer, nil
}

var _ api.VehicleFinishTimer = (*TeslaAPI)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *TeslaAPI) FinishTime() (time.Time, error) {
	res, err := v.dataG()
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Add(time.Duration(res.Response.ChargeState.MinutesToFullCharge) * time.Minute), nil
}

var _ api.VehiclePosition = (*TeslaAPI)(nil)

// Position implements the api.VehiclePosition interface
func (v *TeslaAPI) Position() (float64, float64, error) {
	res, err := v.dataG()
	if err != nil {
                fmt.Println("pos error")
		return 0, 0, err
	}
        fmt.Println("location", res.Response.DriveState.Latitude, res.Response.DriveState.Longitude);
	return res.Response.DriveState.Latitude, res.Response.DriveState.Longitude, nil
}

//var _ api.SocLimiter = (*TeslaAPI)(nil)

// TargetSoc implements the api.SocLimiter interface
//func (v *TeslaAPI) TargetSoc() (float64, error) {
//        fmt.Println("targetsoc")
//	res, err := v.dataG()
//	if err != nil {
//                fmt.Println("targetsoc error")
//		return 0, nil
//	}
//	return float64(res.Response.ChargeState.ChargeLimitSoc), nil
//}

var _ api.CurrentLimiter = (*TeslaAPI)(nil)

// StartCharge implements the api.VehicleChargeController interface

var _ api.Resurrector = (*TeslaAPI)(nil)

func (v *TeslaAPI) WakeUp() error {
	_, err := v.vehicle.Wakeup()
	return v.apiError(err)
}

var _ api.VehicleChargeController = (*TeslaAPI)(nil)

// StartCharge implements the api.VehicleChargeController interface
func (v *TeslaAPI) StartCharge() error {
	err := v.apiError(v.vehicle.StartCharging())
	if err != nil && slices.Contains([]string{"complete", "is_charging"}, err.Error()) {
		return nil
	}
	return err
}

// StopCharge implements the api.VehicleChargeController interface
func (v *TeslaAPI) StopCharge() error {
	err := v.apiError(v.vehicle.StopCharging())

	// ignore sleeping vehicle
	if errors.Is(err, api.ErrAsleep) {
		err = nil
	}

	return err
}
