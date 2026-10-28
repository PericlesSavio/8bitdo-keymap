package dbus

// The daemon runs per user, on the session bus. DBus name elements can't
// start with a digit, hence "EightBitDo".
const (
	BusName       = "io.github.PericlesSavio.EightBitDoCtl"
	InterfaceName = BusName
	ObjectPath    = "/io/github/PericlesSavio/EightBitDoCtl"

	ErrorInvalidArgument = BusName + ".Error.InvalidArgument"
	ErrorFailed          = BusName + ".Error.Failed"
)
