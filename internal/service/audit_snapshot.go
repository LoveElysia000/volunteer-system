package service

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"
	"volunteer-system/internal/model"
)

type auditSnapshotEnvelope struct {
	Scene string          `json:"scene"`
	Data  json.RawMessage `json:"data"`
}

// marshalAuditSnapshot 将任意快照对象序列化为 audit_records 的 JSON 字符串。
// 约定：nil/typed nil 序列化为 "{}"，用于表示空快照。
func marshalAuditSnapshot(payload any) (string, error) {
	if isNilAuditSnapshotPayload(payload) {
		return "{}", nil
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	if string(raw) == "null" {
		return "{}", nil
	}
	return string(raw), nil
}

// marshalAuditSceneSnapshot 将带场景的审核快照封装为包含 scene 和 data 的 JSON 字符串。
func marshalAuditSceneSnapshot(scene string, payload any) (string, error) {
	scene = strings.TrimSpace(scene)
	if scene == "" {
		return "", errors.New("审核场景不能为空")
	}

	rawPayload, err := marshalAuditSnapshot(payload)
	if err != nil {
		return "", err
	}

	envelope := auditSnapshotEnvelope{
		Scene: scene,
		Data:  json.RawMessage(rawPayload),
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// marshalAuditSnapshotPair 序列化旧快照和新快照，返回 old_content 与 new_content。
func marshalAuditSnapshotPair(oldPayload, newPayload any) (string, string, error) {
	oldContent, err := marshalAuditSnapshot(oldPayload)
	if err != nil {
		return "", "", err
	}
	newContent, err := marshalAuditSnapshot(newPayload)
	if err != nil {
		return "", "", err
	}
	return oldContent, newContent, nil
}

// marshalAuditSceneSnapshotPair 序列化带场景的旧新快照，返回 old_content 与 new_content。
func marshalAuditSceneSnapshotPair(scene string, oldPayload, newPayload any) (string, string, error) {
	oldContent, err := marshalAuditSceneSnapshot(scene, oldPayload)
	if err != nil {
		return "", "", err
	}
	newContent, err := marshalAuditSceneSnapshot(scene, newPayload)
	if err != nil {
		return "", "", err
	}
	return oldContent, newContent, nil
}

// parseAuditSnapshotEnvelope 解析审核快照。
// 返回:
// 1. scene
// 2. data（无 envelope 时返回原始内容）
// 3. isEnvelope（是否为 scene+data 封装）
func parseAuditSnapshotEnvelope(raw string) (string, json.RawMessage, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil, false, errors.New("审核内容为空")
	}

	var envelope auditSnapshotEnvelope
	if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
		return "", nil, false, err
	}

	scene := strings.TrimSpace(envelope.Scene)
	if scene == "" {
		return "", json.RawMessage(trimmed), false, nil
	}

	data := envelope.Data
	if len(data) == 0 || string(data) == "null" {
		data = json.RawMessage("{}")
	}
	return scene, data, true, nil
}

// isNilAuditSnapshotPayload 判断审核快照载荷是否为 nil 或 typed nil。
func isNilAuditSnapshotPayload(payload any) bool {
	if payload == nil {
		return true
	}
	v := reflect.ValueOf(payload)
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}

// buildPendingCreateAuditRecordByModel 统一创建“新增”审核记录，快照使用 model 结构。
func buildPendingCreateAuditRecordByModel(
	targetType int32,
	creatorID int64,
	newModelSnapshot any,
	auditTime time.Time,
) (*model.AuditRecord, error) {
	if err := ensureModelSnapshot(newModelSnapshot); err != nil {
		return nil, err
	}
	return buildPendingAuditRecord(
		targetType,
		model.OperationTypeCreate,
		0,
		creatorID,
		nil,
		newModelSnapshot,
		auditTime,
	)
}

// buildPendingDeleteAuditRecordByModel 统一创建“删除”审核记录，快照使用 model 结构。
func buildPendingDeleteAuditRecordByModel(
	targetType int32,
	targetID int64,
	creatorID int64,
	oldModelSnapshot any,
	newModelSnapshot any,
	auditTime time.Time,
) (*model.AuditRecord, error) {
	if err := ensureModelSnapshot(oldModelSnapshot); err != nil {
		return nil, err
	}
	if err := ensureModelSnapshot(newModelSnapshot); err != nil {
		return nil, err
	}
	return buildPendingAuditRecord(
		targetType,
		model.OperationTypeDelete,
		targetID,
		creatorID,
		oldModelSnapshot,
		newModelSnapshot,
		auditTime,
	)
}

// buildPendingUpdateAuditRecordByPatch 统一创建“更新”审核记录，快照使用 patch/DTO 结构。
func buildPendingUpdateAuditRecordByPatch(
	targetType int32,
	targetID int64,
	creatorID int64,
	scene string,
	oldPatchSnapshot any,
	newPatchSnapshot any,
	auditTime time.Time,
) (*model.AuditRecord, error) {
	oldContent, newContent, err := marshalAuditSceneSnapshotPair(scene, oldPatchSnapshot, newPatchSnapshot)
	if err != nil {
		return nil, err
	}
	if auditTime.IsZero() {
		auditTime = time.Now()
	}

	return &model.AuditRecord{
		TargetType:    targetType,
		TargetID:      targetID,
		CreatorID:     creatorID,
		AuditorID:     0,
		OldContent:    oldContent,
		NewContent:    newContent,
		AuditResult:   0,
		RejectReason:  "",
		AuditTime:     auditTime,
		OperationType: model.OperationTypeUpdate,
		Status:        model.AuditStatusPending,
	}, nil
}

// buildPendingAuditRecord 统一构建待审核记录，避免各业务重复拼装公共字段。
func buildPendingAuditRecord(
	targetType int32,
	operationType int32,
	targetID int64,
	creatorID int64,
	oldPayload any,
	newPayload any,
	auditTime time.Time,
) (*model.AuditRecord, error) {
	oldContent, newContent, err := marshalAuditSnapshotPair(oldPayload, newPayload)
	if err != nil {
		return nil, err
	}
	if auditTime.IsZero() {
		auditTime = time.Now()
	}

	return &model.AuditRecord{
		TargetType:    targetType,
		TargetID:      targetID,
		CreatorID:     creatorID,
		AuditorID:     0,
		OldContent:    oldContent,
		NewContent:    newContent,
		AuditResult:   0,
		RejectReason:  "",
		AuditTime:     auditTime,
		OperationType: operationType,
		Status:        model.AuditStatusPending,
	}, nil
}

// ensureModelSnapshot 校验创建/删除审核快照是否使用 model 包中的结构体。
func ensureModelSnapshot(payload any) error {
	if isNilAuditSnapshotPayload(payload) {
		return nil
	}

	t := reflect.TypeOf(payload)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.PkgPath() != "volunteer-system/internal/model" {
		return errors.New("创建/删除审核快照必须使用 model 结构")
	}
	return nil
}
