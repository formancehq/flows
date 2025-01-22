// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

type Account struct {
	Address          string            `json:"address"`
	Metadata         map[string]string `json:"metadata"`
	Volumes          map[string]Volume `json:"volumes,omitempty"`
	EffectiveVolumes map[string]Volume `json:"effectiveVolumes,omitempty"`
}

func (o *Account) GetAddress() string {
	if o == nil {
		return ""
	}
	return o.Address
}

func (o *Account) GetMetadata() map[string]string {
	if o == nil {
		return map[string]string{}
	}
	return o.Metadata
}

func (o *Account) GetVolumes() map[string]Volume {
	if o == nil {
		return nil
	}
	return o.Volumes
}

func (o *Account) GetEffectiveVolumes() map[string]Volume {
	if o == nil {
		return nil
	}
	return o.EffectiveVolumes
}
