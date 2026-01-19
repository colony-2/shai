package shai

// ResourceSet defines runtime resources injected into the sandbox.
type ResourceSet struct {
	Vars         []VarMapping
	Mounts       []Mount
	Calls        []Call
	HTTP         []string
	Ports        []Port
	RootCommands []string
	Options      ResourceOptions
}

// ResourceOptions contains optional resource set configuration.
type ResourceOptions struct {
	Privileged bool
}

// VarMapping defines a host->container variable mapping.
type VarMapping struct {
	Source string
	Target string
}

// Mount describes a host mount.
type Mount struct {
	Source string
	Target string
	Mode   string
}

// Call exposes a host command inside the container.
type Call struct {
	Name        string
	Description string
	Command     string
	AllowedArgs string
}

// Port identifies an allow-listed network endpoint.
type Port struct {
	Host string
	Port int
}
