/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/invite"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
	sys_test_template "github.com/voedger/voedger/pkg/vit/testdata"
)

func TestInvite_BasicUsage(t *testing.T) {
	require := require.New(t)

	host := "host@email.com"
	guest := "guest@email.com"
	wsName := "test_ws"

	cfg := it.NewOwnVITConfig(it.WithApp(istructs.AppQName_test1_app1, it.ProvideApp1,
		it.WithUserLogin(host, "host_pwd"),
		it.WithUserLogin(guest, "guest_pwd"),
		it.WithWorkspaceTemplate(it.QNameApp1_TestWSKind, "test_template", sys_test_template.TestTemplateFS),
		it.WithChildWorkspace(it.QNameApp1_TestWSKind, wsName, "test_template", "", host, map[string]interface{}{"IntFld": 42}),
	))
	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, wsName)
	inviteEmailTemplate := "text:" + strings.Join([]string{
		invite.EmailTemplatePlaceholder_VerificationCode,
		invite.EmailTemplatePlaceholder_InviteID,
		invite.EmailTemplatePlaceholder_WSID,
		invite.EmailTemplatePlaceholder_WSName,
		invite.EmailTemplatePlaceholder_Email,
	}, ";")
	inviteEmailSubject := "you are invited"
	expireDatetime := vit.Now().UnixMilli()
	expireDatetimeStr := strconv.FormatInt(expireDatetime, 10)
	verificationCode := expireDatetimeStr[len(expireDatetimeStr)-6:]
	roles := "initial.Roles"

	findCDocInviteByID := func(inviteID int64, inviteState int32) (entity []interface{}) {
		deadline := time.Now().Add(time.Second * 5)
		for time.Now().Before(deadline) {
			entity = vit.PostWS(ws, "q.sys.Collection", fmt.Sprintf(`
			{"args":{"Schema":"sys.Invite"},
			"elements":[{"fields":[
				"SubjectKind",
				"Login",
				"Email",
				"Roles",
				"ExpireDatetime",
				"VerificationCode",
				"State",
				"Created",
				"Updated",
				"SubjectID",
				"InviteeProfileWSID",
				"sys.ID"
			]}],
			"filters":[{"expr":"eq","args":{"field":"sys.ID","value":%d}}]}`, inviteID)).SectionRow(0)
			if inviteState == int32(entity[6].(float64)) {
				return
			}
		}
		require.FailNow(fmt.Sprintf("invite [%d] is not in required state [%d] it has state [%d]", inviteID, inviteState, int32(entity[0].(float64))))
		return
	}

	findCDocSubjectByLogin := func(login string) []interface{} {
		return vit.PostWS(ws, "q.sys.Collection", fmt.Sprintf(`
			{"args":{"Schema":"sys.Subject"},
			"elements":[{"fields":[
				"Login",
				"SubjectKind",
				"Roles",
				"sys.ID",
				"sys.IsActive"
			]}],
			"filters":[{"expr":"eq","args":{"field":"Login","value":"%s"}}]}`, login)).SectionRow(0)
	}

	//Invite guest
	inviteID := InitiateInvitationByEMail(vit, ws, expireDatetime, guest, roles, inviteEmailTemplate, inviteEmailSubject)

	cDocInvite := findCDocInviteByID(inviteID, invite.State_ToBeInvited)
	require.Equal(guest, cDocInvite[1])
	require.Equal(guest, cDocInvite[2])
	require.Equal(roles, cDocInvite[3])
	require.Equal(float64(expireDatetime), cDocInvite[4])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[7])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	//Check that emails were send and guest is invited
	cDocInvite = findCDocInviteByID(inviteID, invite.State_Invited)

	require.Equal(verificationCode, cDocInvite[5])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	message := vit.CaptureEmail()
	ss := strings.Split(message.Body, ";")
	require.Equal(inviteEmailSubject, message.Subject)
	require.Equal(it.TestSMTPCfg.GetFrom(), message.From)
	require.Equal([]string{guest}, message.To)
	require.Equal(verificationCode, ss[0])
	require.Equal(strconv.FormatInt(inviteID, 10), ss[1])
	require.Equal(strconv.FormatInt(int64(ws.WSID), 10), ss[2])
	require.Equal(wsName, ss[3])
	require.Equal(guest, ss[4])

	//Join workspace
	InitiateJoinWorkspace(vit, ws, inviteID, guest, verificationCode)

	findCDocInviteByID(inviteID, invite.State_ToBeJoined)

	cDocInvite = findCDocInviteByID(inviteID, invite.State_Joined)
	require.Equal(float64(vit.GetPrincipal(istructs.AppQName_test1_app1, guest).ProfileWSID), cDocInvite[10])
	require.Equal(float64(istructs.SubjectKind_User), cDocInvite[0])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	cDocJoinedWorkspace := FindCDocJoinedWorkspaceByInvitingWorkspaceWSIDAndLogin(vit, ws.WSID, guest)
	require.Equal(roles, cDocJoinedWorkspace.roles)
	require.Equal(wsName, cDocJoinedWorkspace.wsName)

	cDocSubject := findCDocSubjectByLogin(guest)

	require.Equal(guest, cDocSubject[0])
	require.Equal(float64(istructs.SubjectKind_User), cDocSubject[1])
	require.Equal(roles, cDocSubject[2])
}

func TestInvite(t *testing.T) {
	//t.Run("None -> Invited -> Joined", func(t *testing.T) {
	//
	//})
	//t.Run("None -> Invited -> Joined -> Cancelled", func(t *testing.T) {
	//
	//})
	//t.Run("BasicUsage", func(t *testing.T) {
	//
	//})
	t.Run("Update roles", func(t *testing.T) {

	})
	t.Run("Left workspace", func(t *testing.T) {

	})
	t.Run("Cancel from workspace", func(t *testing.T) {

	})
	//TODO extra test
	t.Run("Cancel invite", func(t *testing.T) {

	})
	//
	//host := "host@email.com"
	//guest1 := "guest1@email.com"
	////guest2 := "guest2@email.com"
	////guest2 := "guest2@email.com"
	//wsName1 := "test_ws1"
	//
	//cfg := it.NewOwnVITConfig(it.WithApp(istructs.AppQName_test1_app1, it.ProvideApp1,
	//	it.WithUserLogin(host, "host_pwd"),
	//	it.WithUserLogin(guest1, "guest1_pwd"),
	//	it.WithWorkspaceTemplate(it.QNameApp1_TestWSKind, "test_template", sys_test_template.TestTemplateFS),
	//	it.WithChildWorkspace(it.QNameApp1_TestWSKind, wsName1, "test_template", "", host, map[string]interface{}{"IntFld": 1}),
	//))
	//vit := it.NewVIT(t, &cfg)
	//defer vit.TearDown()
	//
	//inviteEmailTemplate := "text:" + strings.Join([]string{
	//	invite.EmailTemplatePlaceholder_VerificationCode,
	//	invite.EmailTemplatePlaceholder_InviteID,
	//	invite.EmailTemplatePlaceholder_WSID,
	//	invite.EmailTemplatePlaceholder_WSName,
	//	invite.EmailTemplatePlaceholder_Email,
	//}, ";")
	//inviteEmailSubject := "you are invited"
	//expireDatetime := vit.Now().UnixMilli()
	//expireDatetimeStr := strconv.FormatInt(expireDatetime, 10)
	//verificationCode := expireDatetimeStr[len(expireDatetimeStr)-6:]
	//roles := "initial.Roles"
	//
	//joinWorkspace := func(ws *it.AppWorkspace, email string) {
	//	inviteID := InitiateInvitationByEMail(vit, ws, expireDatetime, email, roles, inviteEmailTemplate, inviteEmailSubject)
	//	vit.CaptureEmail()
	//
	//}
	//
	//t.Run("Update roles", func(t *testing.T) {
	//
	//})
}

func TestInvite_BasicUsage2(t *testing.T) {
	//TODO Daniil fix it
	t.Skip("Daniil fix it https://dev.untill.com/projects/#!639145")
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	wsName := "test_ws"
	ws := vit.WS(istructs.AppQName_test1_app1, wsName)
	inviteEmailTemplate := "text:" + strings.Join([]string{
		invite.EmailTemplatePlaceholder_VerificationCode,
		invite.EmailTemplatePlaceholder_InviteID,
		invite.EmailTemplatePlaceholder_WSID,
		invite.EmailTemplatePlaceholder_WSName,
		invite.EmailTemplatePlaceholder_Email,
	}, ";")
	inviteEmailSubject := "you are invited"
	updateRolesEmailTemplate := "text:" + invite.EmailTemplatePlaceholder_Roles
	updateRolesEmailSubject := "your roles are updated"
	expireDatetime := vit.Now().UnixMilli()
	expireDatetimeStr := strconv.FormatInt(expireDatetime, 10)
	verificationCode := expireDatetimeStr[len(expireDatetimeStr)-6:]
	initialRoles := "initial.Roles"
	updatedRoles := "updated.Roles"

	initiateUpdateInviteRoles := func(inviteID int64) {
		vit.PostWS(ws, "c.sys.InitiateUpdateInviteRoles", fmt.Sprintf(`{"args":{"InviteID":%d,"Roles":"%s","EmailTemplate":"%s","EmailSubject":"%s"}}`, inviteID, updatedRoles, updateRolesEmailTemplate, updateRolesEmailSubject))
	}

	findCDocInviteByID := func(inviteID int64) []interface{} {
		return vit.PostWS(ws, "q.sys.Collection", fmt.Sprintf(`
			{"args":{"Schema":"sys.Invite"},
			"elements":[{"fields":[
				"SubjectKind",
				"Login",
				"Email",
				"Roles",
				"ExpireDatetime",
				"VerificationCode",
				"State",
				"Created",
				"Updated",
				"SubjectID",
				"InviteeProfileWSID",
				"sys.ID"
			]}],
			"filters":[{"expr":"eq","args":{"field":"sys.ID","value":%d}}]}`, inviteID)).SectionRow(0)
	}

	findCDocSubjectByLogin := func(login string) []interface{} {
		return vit.PostWS(ws, "q.sys.Collection", fmt.Sprintf(`
			{"args":{"Schema":"sys.Subject"},
			"elements":[{"fields":[
				"Login",
				"SubjectKind",
				"Roles",
				"sys.ID",
				"sys.IsActive"
			]}],
			"filters":[{"expr":"eq","args":{"field":"Login","value":"%s"}}]}`, login)).SectionRow(0)
	}

	//Invite existing users
	inviteID := InitiateInvitationByEMail(vit, ws, expireDatetime, it.TestEmail, initialRoles, inviteEmailTemplate, inviteEmailSubject)
	inviteID2 := InitiateInvitationByEMail(vit, ws, expireDatetime, it.TestEmail2, initialRoles, inviteEmailTemplate, inviteEmailSubject)
	inviteID3 := InitiateInvitationByEMail(vit, ws, expireDatetime, it.TestEmail3, initialRoles, inviteEmailTemplate, inviteEmailSubject)

	WaitForInviteState(vit, ws, invite.State_ToBeInvited, inviteID)
	WaitForInviteState(vit, ws, invite.State_ToBeInvited, inviteID2)
	WaitForInviteState(vit, ws, invite.State_ToBeInvited, inviteID3)

	cDocInvite := findCDocInviteByID(inviteID)

	require.Equal(it.TestEmail, cDocInvite[1])
	require.Equal(it.TestEmail, cDocInvite[2])
	require.Equal(initialRoles, cDocInvite[3])
	require.Equal(float64(expireDatetime), cDocInvite[4])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[7])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	//Check that emails were send
	require.Equal(verificationCode, strings.Split(vit.CaptureEmail().Body, ";")[0])
	require.Equal(verificationCode, strings.Split(vit.CaptureEmail().Body, ";")[0])

	message := vit.CaptureEmail()
	ss := strings.Split(message.Body, ";")
	require.Equal(inviteEmailSubject, message.Subject)
	require.Equal(it.TestSMTPCfg.GetFrom(), message.From)
	require.Equal([]string{it.TestEmail3}, message.To)
	require.Equal(verificationCode, ss[0])
	require.Equal(strconv.FormatInt(inviteID3, 10), ss[1])
	require.Equal(strconv.FormatInt(int64(ws.WSID), 10), ss[2])
	require.Equal(wsName, ss[3])
	require.Equal(it.TestEmail3, ss[4])

	WaitForInviteState(vit, ws, invite.State_Invited, inviteID)
	WaitForInviteState(vit, ws, invite.State_Invited, inviteID2)
	WaitForInviteState(vit, ws, invite.State_Invited, inviteID3)

	cDocInvite = findCDocInviteByID(inviteID2)

	require.Equal(verificationCode, cDocInvite[5])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	//Cancel then invite it again (inviteID3)
	vit.PostWS(ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID3))
	WaitForInviteState(vit, ws, invite.State_Cancelled, inviteID3)
	InitiateInvitationByEMail(vit, ws, expireDatetime, it.TestEmail3, initialRoles, inviteEmailTemplate, inviteEmailSubject)
	WaitForInviteState(vit, ws, invite.State_ToBeInvited, inviteID3)
	_ = vit.CaptureEmail()
	WaitForInviteState(vit, ws, invite.State_Invited, inviteID3)

	//Join workspaces
	InitiateJoinWorkspace(vit, ws, inviteID, it.TestEmail, verificationCode)
	InitiateJoinWorkspace(vit, ws, inviteID2, it.TestEmail2, verificationCode)

	WaitForInviteState(vit, ws, invite.State_ToBeJoined, inviteID)
	WaitForInviteState(vit, ws, invite.State_ToBeJoined, inviteID2)

	cDocInvite = findCDocInviteByID(inviteID2)

	require.Equal(float64(vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail2).ProfileWSID), cDocInvite[10])
	require.Equal(float64(istructs.SubjectKind_User), cDocInvite[0])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	WaitForInviteState(vit, ws, invite.State_Joined, inviteID)
	WaitForInviteState(vit, ws, invite.State_Joined, inviteID2)

	cDocJoinedWorkspace := FindCDocJoinedWorkspaceByInvitingWorkspaceWSIDAndLogin(vit, ws.WSID, it.TestEmail2)

	require.Equal(initialRoles, cDocJoinedWorkspace.roles)
	require.Equal(wsName, cDocJoinedWorkspace.wsName)

	cDocSubject := findCDocSubjectByLogin(it.TestEmail)

	require.Equal(it.TestEmail, cDocSubject[0])
	require.Equal(float64(istructs.SubjectKind_User), cDocSubject[1])
	require.Equal(initialRoles, cDocSubject[2])

	//Update roles
	initiateUpdateInviteRoles(inviteID)
	initiateUpdateInviteRoles(inviteID2)

	WaitForInviteState(vit, ws, invite.State_ToUpdateRoles, inviteID)
	WaitForInviteState(vit, ws, invite.State_ToUpdateRoles, inviteID2)
	cDocInvite = findCDocInviteByID(inviteID)

	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	//Check that emails were send
	require.Equal(updatedRoles, vit.CaptureEmail().Body)
	message = vit.CaptureEmail()
	require.Equal(updateRolesEmailSubject, message.Subject)
	require.Equal(it.TestSMTPCfg.GetFrom(), message.From)
	require.Equal([]string{it.TestEmail2}, message.To)
	require.Equal(updatedRoles, message.Body)

	cDocSubject = findCDocSubjectByLogin(it.TestEmail2)

	require.Equal(updatedRoles, cDocSubject[2])

	//TODO Denis how to get WS by login? I want to check sys.JoinedWorkspace

	WaitForInviteState(vit, ws, invite.State_Joined, inviteID)
	WaitForInviteState(vit, ws, invite.State_Joined, inviteID2)

	//Cancel accepted invite
	vit.PostWS(ws, "c.sys.InitiateCancelAcceptedInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID))

	WaitForInviteState(vit, ws, invite.State_ToBeCancelled, inviteID)

	cDocInvite = findCDocInviteByID(inviteID)

	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	WaitForInviteState(vit, ws, invite.State_Cancelled, inviteID)

	cDocSubject = findCDocSubjectByLogin(it.TestEmail)

	require.False(cDocSubject[4].(bool))

	cDocInvite = findCDocInviteByID(inviteID)

	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	//Leave workspace
	vit.PostWS(ws, "c.sys.InitiateLeaveWorkspace", "{}", coreutils.WithAuthorizeBy(vit.GetPrincipal(ws.Owner.AppQName, it.TestEmail2).Token))

	WaitForInviteState(vit, ws, invite.State_ToBeLeft, inviteID2)

	cDocInvite = findCDocInviteByID(inviteID2)

	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	WaitForInviteState(vit, ws, invite.State_Left, inviteID2)

	cDocSubject = findCDocSubjectByLogin(it.TestEmail2)

	require.False(cDocSubject[4].(bool))

	//TODO check InviteeProfile joined workspace
}

func TestCancelSentInvite(t *testing.T) {
	//TODO Fix it Daniil
	t.Skip("Fix it Daniil")
	//require := require.New(t)
	//vit := it.NewVIT(t, &it.SharedConfig_Simple)
	//defer vit.TearDown()
	//ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	//
	//t.Run("Should be ok", func(t *testing.T) {
	//	body := `{"args":{"Email":"user@acme.com","Roles":"trolles","ExpireDatetime":1674751138000,"NewLoginEmailTemplate":"text:","ExistingLoginEmailTemplate":"text:"}}`
	//	inviteID := vit.PostWS(ws, "c.sys.InitiateInvitationByEMail", body).NewID()
	//	//Read it for successful vit tear down
	//	_ = vit.ExpectEmail().Capture()
	//
	//	vit.PostWS(ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID))
	//})
	//t.Run("Should be not ok", func(t *testing.T) {
	//	resp := vit.PostWS(ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, -100), coreutils.Expect400())
	//
	//	require.Equal(coreutils.NewHTTPError(http.StatusBadRequest, invite.errInviteNotExists), resp.SysError)
	//})
}
