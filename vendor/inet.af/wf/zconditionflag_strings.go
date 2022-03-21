// Code generated by "stringer -output=zconditionflag_strings.go -type=ConditionFlag -trimprefix=ConditionFlag"; DO NOT EDIT.

package wf

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ConditionFlagIsLoopback-1]
	_ = x[ConditionFlagIsIPSecSecured-2]
	_ = x[ConditionFlagIsReauthorize-4]
	_ = x[ConditionFlagIsWildcardBind-8]
	_ = x[ConditionFlagIsRawEndpoint-16]
	_ = x[ConditionFlagIsFragmant-32]
	_ = x[ConditionFlagIsFragmantGroup-64]
	_ = x[ConditionFlagIsIPSecNATTReclassify-128]
	_ = x[ConditionFlagIsRequiresALEClassify-256]
	_ = x[ConditionFlagIsImplicitBind-512]
	_ = x[ConditionFlagIsReassembled-1024]
	_ = x[ConditionFlagIsNameAppSpecified-16384]
	_ = x[ConditionFlagIsPromiscuous-32768]
	_ = x[ConditionFlagIsAuthFW-65536]
	_ = x[ConditionFlagIsReclassify-131072]
	_ = x[ConditionFlagIsOutboundPassThru-262144]
	_ = x[ConditionFlagIsInboundPassThru-524288]
	_ = x[ConditionFlagIsConnectionRedirected-1048576]
}

const _ConditionFlag_name = "IsLoopbackIsIPSecSecuredIsReauthorizeIsWildcardBindIsRawEndpointIsFragmantIsFragmantGroupIsIPSecNATTReclassifyIsRequiresALEClassifyIsImplicitBindIsReassembledIsNameAppSpecifiedIsPromiscuousIsAuthFWIsReclassifyIsOutboundPassThruIsInboundPassThruIsConnectionRedirected"

var _ConditionFlag_map = map[ConditionFlag]string{
	1:       _ConditionFlag_name[0:10],
	2:       _ConditionFlag_name[10:24],
	4:       _ConditionFlag_name[24:37],
	8:       _ConditionFlag_name[37:51],
	16:      _ConditionFlag_name[51:64],
	32:      _ConditionFlag_name[64:74],
	64:      _ConditionFlag_name[74:89],
	128:     _ConditionFlag_name[89:110],
	256:     _ConditionFlag_name[110:131],
	512:     _ConditionFlag_name[131:145],
	1024:    _ConditionFlag_name[145:158],
	16384:   _ConditionFlag_name[158:176],
	32768:   _ConditionFlag_name[176:189],
	65536:   _ConditionFlag_name[189:197],
	131072:  _ConditionFlag_name[197:209],
	262144:  _ConditionFlag_name[209:227],
	524288:  _ConditionFlag_name[227:244],
	1048576: _ConditionFlag_name[244:266],
}

func (i ConditionFlag) String() string {
	if str, ok := _ConditionFlag_map[i]; ok {
		return str
	}
	return "ConditionFlag(" + strconv.FormatInt(int64(i), 10) + ")"
}