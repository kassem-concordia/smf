package context

import (
	"github.com/free5gc/aper"
	"github.com/free5gc/nas/nasType"
	"github.com/free5gc/ngap/ngapType"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/smf/internal/util"
)

// QoSFlow  - Policy and Charging Rule

type QoSFlowState int

const (
	QoSFlowUnset QoSFlowState = iota
	QoSFlowSet
	QoSFlowToBeModify
)

type QoSFlow struct {
	QFI        uint8
	QoSProfile *models.QosData
	State      QoSFlowState
	AltQosProfiles []*models.QosData //kassem
}

func NewQoSFlow(qfi uint8, qosModel *models.QosData) *QoSFlow {
	if qosModel == nil {
		return nil
	}
	qos := &QoSFlow{
		QFI:        qfi,
		QoSProfile: qosModel,
		State:      QoSFlowUnset,
	}
	return qos
}

func (q *QoSFlow) GetQFI() uint8 {
	return q.QFI
}

func (q *QoSFlow) Get5QI() uint8 {
	return uint8(q.QoSProfile.Var5qi)
}

func (q *QoSFlow) GetQoSProfile() *models.QosData {
	return q.QoSProfile
}

func (q *QoSFlow) IsGBRFlow() bool {
	return isGBRFlow(q.QoSProfile)
}

func (q *QoSFlow) BuildNasQoSDesc(opCode nasType.QoSFlowOperationCode) (nasType.QoSFlowDesc, error) {
	qosDesc := nasType.QoSFlowDesc{}
	qosDesc.QFI = q.GetQFI()
	qosDesc.OperationCode = opCode
	parameter := new(nasType.QoSFlow5QI)
	parameter.FiveQI = uint8(q.QoSProfile.Var5qi)
	qosDesc.Parameters = append(qosDesc.Parameters, parameter)

	if q.IsGBRFlow() && q.QoSProfile != nil {
		gbrDlParameter := new(nasType.QoSFlowGFBRDownlink)
		gbrDlParameter.Unit = nasType.QoSFlowBitRateUnit1Mbps
		gbrDlParameter.Value = util.BitRateTombps(q.QoSProfile.GbrDl)
		qosDesc.Parameters = append(qosDesc.Parameters, gbrDlParameter)
		gbrUlParameter := new(nasType.QoSFlowGFBRUplink)
		gbrUlParameter.Unit = nasType.QoSFlowBitRateUnit1Mbps
		gbrUlParameter.Value = util.BitRateTombps(q.QoSProfile.GbrUl)
		qosDesc.Parameters = append(qosDesc.Parameters, gbrUlParameter)
		mbrDlParameter := new(nasType.QoSFlowMFBRDownlink)
		mbrDlParameter.Unit = nasType.QoSFlowBitRateUnit1Mbps
		mbrDlParameter.Value = util.BitRateTombps(q.QoSProfile.MaxbrDl)
		qosDesc.Parameters = append(qosDesc.Parameters, mbrDlParameter)
		mbrUlParameter := new(nasType.QoSFlowMFBRUplink)
		mbrUlParameter.Unit = nasType.QoSFlowBitRateUnit1Mbps
		mbrUlParameter.Value = util.BitRateTombps(q.QoSProfile.MaxbrUl)
		qosDesc.Parameters = append(qosDesc.Parameters, mbrUlParameter)
	}
	return qosDesc, nil
}

func buildArpFromModels(arp *models.Arp) (int64, aper.Enumerated, aper.Enumerated) {
	if arp == nil {
		return 0, 0, 0
	}
	var arpPriorityLevel int64
	var arpPreEmptionCapability aper.Enumerated
	var arpPreEmptionVulnerability aper.Enumerated

	arpPriorityLevel = int64(arp.PriorityLevel)
	switch arp.PreemptCap {
	case models.PreemptionCapability_NOT_PREEMPT:
		arpPreEmptionCapability = ngapType.PreEmptionCapabilityPresentShallNotTriggerPreEmption
	case models.PreemptionCapability_MAY_PREEMPT:
		arpPreEmptionCapability = ngapType.PreEmptionCapabilityPresentMayTriggerPreEmption
	default:
		arpPreEmptionCapability = ngapType.PreEmptionCapabilityPresentShallNotTriggerPreEmption
	}
	switch arp.PreemptVuln {
	case models.PreemptionVulnerability_NOT_PREEMPTABLE:
		arpPreEmptionVulnerability = ngapType.PreEmptionVulnerabilityPresentNotPreEmptable
	case models.PreemptionVulnerability_PREEMPTABLE:
		arpPreEmptionVulnerability = ngapType.PreEmptionVulnerabilityPresentPreEmptable
	default:
		arpPreEmptionVulnerability = ngapType.PreEmptionVulnerabilityPresentNotPreEmptable
	}

	return arpPriorityLevel, arpPreEmptionCapability, arpPreEmptionVulnerability
}

func buildGBRQosInformationFromModel(qos *models.QosData) *ngapType.GBRQosInformation {
	if qos == nil {
		return nil
	}
	gbrInfo := &ngapType.GBRQosInformation{
		MaximumFlowBitRateDL:    util.StringToBitRate(qos.MaxbrDl),
		MaximumFlowBitRateUL:    util.StringToBitRate(qos.MaxbrUl),
		GuaranteedFlowBitRateDL: util.StringToBitRate(qos.GbrDl),
		GuaranteedFlowBitRateUL: util.StringToBitRate(qos.GbrUl),
	}
	if qos.Qnc { //kassem
		notifCtrl := ngapType.NotificationControl{} 
		notifCtrl.Value = ngapType.NotificationControlPresentNotificationRequested 
		gbrInfo.NotificationControl = &notifCtrl 
	} //kassem

	return gbrInfo
}

func buildAltQoSParaSetList(altProfiles []*models.QosData) *ngapType.AlternativeQoSParaSetList { //kassem
	if len(altProfiles) == 0 { 
		return nil 
	} 
	list := &ngapType.AlternativeQoSParaSetList{} 
	for i, alt := range altProfiles { 
		if alt == nil { 
			continue 
		} 
		if i >= 8 { 
			break 
		} 
		item := ngapType.AlternativeQoSParaSetItem{ 
			AlternativeQoSParaSetIndex: ngapType.AlternativeQoSParaSetIndex{ 
				Value: int64(i + 1), 
			},
		} 
		// Populate GBR values for this alternative profile 
		if alt.GbrDl != "" || alt.GbrUl != "" { 
			item.GuaranteedFlowBitRateDL = new(ngapType.BitRate) 
			item.GuaranteedFlowBitRateDL.Value = util.StringToBitRate(alt.GbrDl).Value 
			item.GuaranteedFlowBitRateUL = new(ngapType.BitRate) 
			item.GuaranteedFlowBitRateUL.Value = util.StringToBitRate(alt.GbrUl).Value 
		} 
		if alt.MaxbrDl != "" || alt.MaxbrUl != "" { 
			item.MaximumFlowBitRateDL = new(ngapType.BitRate) 
			item.MaximumFlowBitRateDL.Value = util.StringToBitRate(alt.MaxbrDl).Value 
			item.MaximumFlowBitRateUL = new(ngapType.BitRate) 
			item.MaximumFlowBitRateUL.Value = util.StringToBitRate(alt.MaxbrUl).Value 
		} 
		list.List = append(list.List, item) 
	} 
	if len(list.List) == 0 { 
		return nil 
	}
	return list
}//kaSSEM


func (q *QoSFlow) BuildNgapQosFlowSetupRequestItem() (ngapType.QosFlowSetupRequestItem, error) {
	qosDesc := ngapType.QosFlowSetupRequestItem{}

	qosDesc.QosFlowIdentifier = ngapType.QosFlowIdentifier{
		Value: int64(q.GetQFI()),
	}

	parameter := ngapType.QosFlowLevelQosParameters{}
	parameter.QosCharacteristics = ngapType.QosCharacteristics{
		Present: ngapType.QosCharacteristicsPresentNonDynamic5QI,
		NonDynamic5QI: &ngapType.NonDynamic5QIDescriptor{
			FiveQI: ngapType.FiveQI{
				Value: int64(q.Get5QI()),
			},
		},
	}

	if q.IsGBRFlow() {
		parameter.GBRQosInformation = buildGBRQosInformationFromModel(q.QoSProfile)

		// Attach alternative QoS parameter sets when present
		if parameter.GBRQosInformation != nil && len(q.AltQosProfiles) > 0 { //kassem
			parameter.GBRQosInformation.AlternativeQoSParaSetList = buildAltQoSParaSetList(q.AltQosProfiles)
		} //kassem
	}

	var arpPriorityLevel int64
	var arpPreEmptionCapability aper.Enumerated
	var arpPreEmptionVulnerability aper.Enumerated
	if arp := q.QoSProfile.Arp; arp != nil {
		arpPriorityLevel,
			arpPreEmptionCapability,
			arpPreEmptionVulnerability = buildArpFromModels(arp)
	} else {
		// TODO: should get value from PCF
		arpPriorityLevel = 8
		arpPreEmptionCapability = ngapType.PreEmptionCapabilityPresentShallNotTriggerPreEmption
		arpPreEmptionVulnerability = ngapType.PreEmptionVulnerabilityPresentNotPreEmptable
	}

	parameter.AllocationAndRetentionPriority = ngapType.AllocationAndRetentionPriority{
		PriorityLevelARP: ngapType.PriorityLevelARP{
			Value: arpPriorityLevel,
		},
		PreEmptionCapability: ngapType.PreEmptionCapability{
			Value: arpPreEmptionCapability,
		},
		PreEmptionVulnerability: ngapType.PreEmptionVulnerability{
			Value: arpPreEmptionVulnerability,
		},
	}

	qosDesc.QosFlowLevelQosParameters = parameter

	return qosDesc, nil
}

func (q *QoSFlow) BuildNgapQosFlowAddOrModifyRequestItem() (ngapType.QosFlowAddOrModifyRequestItem, error) {
	qosDesc := ngapType.QosFlowAddOrModifyRequestItem{}

	qosDesc.QosFlowIdentifier = ngapType.QosFlowIdentifier{
		Value: int64(q.GetQFI()),
	}

	parameter := ngapType.QosFlowLevelQosParameters{}
	parameter.QosCharacteristics = ngapType.QosCharacteristics{
		Present: ngapType.QosCharacteristicsPresentNonDynamic5QI,
		NonDynamic5QI: &ngapType.NonDynamic5QIDescriptor{
			FiveQI: ngapType.FiveQI{
				Value: int64(q.Get5QI()),
			},
		},
	}

	if q.IsGBRFlow() {
		parameter.GBRQosInformation = buildGBRQosInformationFromModel(q.QoSProfile)
	}

	var arpPriorityLevel int64
	var arpPreEmptionCapability aper.Enumerated
	var arpPreEmptionVulnerability aper.Enumerated
	if arp := q.QoSProfile.Arp; arp != nil {
		arpPriorityLevel,
			arpPreEmptionCapability,
			arpPreEmptionVulnerability = buildArpFromModels(arp)
	} else {
		// TODO: should get value from PCF
		arpPriorityLevel = 8
		arpPreEmptionCapability = ngapType.PreEmptionCapabilityPresentShallNotTriggerPreEmption
		arpPreEmptionVulnerability = ngapType.PreEmptionVulnerabilityPresentNotPreEmptable
	}

	parameter.AllocationAndRetentionPriority = ngapType.AllocationAndRetentionPriority{
		PriorityLevelARP: ngapType.PriorityLevelARP{
			Value: arpPriorityLevel,
		},
		PreEmptionCapability: ngapType.PreEmptionCapability{
			Value: arpPreEmptionCapability,
		},
		PreEmptionVulnerability: ngapType.PreEmptionVulnerability{
			Value: arpPreEmptionVulnerability,
		},
	}

	qosDesc.QosFlowLevelQosParameters = &parameter

	return qosDesc, nil
}
