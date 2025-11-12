package locales

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed ru.json
var rawRU []byte

type Messages struct {
	MainMenuText                         string   `json:"main_menu_text"`
	MainMenuButtons                      []string `json:"main_menu_buttons"`
	CustomerServiceUnavailableText       string   `json:"customer_service_unavailable_text"`
	CustomerLookupErrorText              string   `json:"customer_lookup_error_text"`
	CustomerFormIntroText                string   `json:"customer_form_intro_text"`
	CustomerSummaryTitle                 string   `json:"customer_summary_title"`
	CustomerSummaryTemplate              string   `json:"customer_summary_template"`
	CustomerTypePrompt                   string   `json:"customer_type_prompt"`
	CustomerTypeIndividualButton         string   `json:"customer_type_individual_button"`
	CustomerTypeBusinessButton           string   `json:"customer_type_business_button"`
	CustomerTypeIndividualLabel          string   `json:"customer_type_individual_label"`
	CustomerTypeBusinessLabel            string   `json:"customer_type_business_label"`
	CustomerNamePrompt                   string   `json:"customer_name_prompt"`
	CustomerNameRetryText                string   `json:"customer_name_retry_text"`
	CustomerNamePromptIndividual         string   `json:"customer_name_prompt_individual"`
	CustomerNamePromptBusiness           string   `json:"customer_name_prompt_business"`
	CustomerNameRetryIndividual          string   `json:"customer_name_retry_individual"`
	CustomerNameRetryBusiness            string   `json:"customer_name_retry_business"`
	CustomerAboutPrompt                  string   `json:"customer_about_prompt"`
	CustomerAboutRetryText               string   `json:"customer_about_retry_text"`
	CustomerAboutPromptIndividual        string   `json:"customer_about_prompt_individual"`
	CustomerAboutPromptBusiness          string   `json:"customer_about_prompt_business"`
	CustomerAboutRetryIndividual         string   `json:"customer_about_retry_individual"`
	CustomerAboutRetryBusiness           string   `json:"customer_about_retry_business"`
	CustomerCreateSuccessText            string   `json:"customer_create_success_text"`
	CustomerUpdateSuccessText            string   `json:"customer_update_success_text"`
	CustomerSaveErrorText                string   `json:"customer_save_error_text"`
	CustomerManageCreateButton           string   `json:"customer_manage_create_button"`
	CustomerManageUpdateButton           string   `json:"customer_manage_update_button"`
	CustomerManageDeleteButton           string   `json:"customer_manage_delete_button"`
	CustomerManageBackButton             string   `json:"customer_manage_back_button"`
	CustomerManageTasksButton            string   `json:"customer_manage_tasks_button"`
	CustomerManageCreateTaskButton       string   `json:"customer_manage_create_task_button"`
	CustomerTasksListText                string   `json:"customer_tasks_list_text"`
	CustomerCreateTaskPlaceholderText    string   `json:"customer_create_task_placeholder_text"`
	CustomerTasksEmptyText               string   `json:"customer_tasks_empty_text"`
	CustomerTaskItemTemplate             string   `json:"customer_task_item_template"`
	CustomerTasksPrevButton              string   `json:"customer_tasks_prev_button"`
	CustomerTasksNextButton              string   `json:"customer_tasks_next_button"`
	CustomerTasksPageFooter              string   `json:"customer_tasks_page_footer"`
	CustomerTaskRewardDescription        string   `json:"customer_task_reward_description"`
	CustomerTaskDetailFormat             string   `json:"customer_task_detail_format"`
	CustomerTaskDetailLocation           string   `json:"customer_task_detail_location"`
	CustomerTaskDetailReward             string   `json:"customer_task_detail_reward"`
	CustomerTaskDetailNoReward           string   `json:"customer_task_detail_no_reward"`
	CustomerTaskDetailVolunteersOne      string   `json:"customer_task_detail_volunteers_one"`
	CustomerTaskDetailVolunteersMany     string   `json:"customer_task_detail_volunteers_many"`
	CustomerTaskDetailCreatedAt          string   `json:"customer_task_detail_created_at"`
	CustomerTaskAssignmentsEmptyText     string   `json:"customer_task_assignments_empty_text"`
	VolunteerMenuIntro                   string   `json:"volunteer_menu_intro"`
	VolunteerMenuOnDemandButton          string   `json:"volunteer_menu_on_demand_button"`
	VolunteerMenuTasksButton             string   `json:"volunteer_menu_tasks_button"`
	VolunteerMenuProfileButton           string   `json:"volunteer_menu_profile_button"`
	VolunteerMenuBackButton              string   `json:"volunteer_menu_back_button"`
	VolunteerMenuMainButton              string   `json:"volunteer_menu_main_button"`
	VolunteerOnDemandPlaceholder         string   `json:"volunteer_on_demand_placeholder"`
	VolunteerTasksPlaceholder            string   `json:"volunteer_tasks_placeholder"`
	VolunteerTasksUnavailableText        string   `json:"volunteer_tasks_unavailable_text"`
	VolunteerTasksErrorText              string   `json:"volunteer_tasks_error_text"`
	VolunteerTasksEmptyText              string   `json:"volunteer_tasks_empty_text"`
	VolunteerTasksFilterAllButton        string   `json:"volunteer_tasks_filter_all_button"`
	VolunteerTasksFilterRewardButton     string   `json:"volunteer_tasks_filter_reward_button"`
	VolunteerTasksFilterTeamButton       string   `json:"volunteer_tasks_filter_team_button"`
	VolunteerTasksFilterOnlineButton     string   `json:"volunteer_tasks_filter_online_button"`
	VolunteerTasksFilterAllLabel         string   `json:"volunteer_tasks_filter_all_label"`
	VolunteerTasksFilterRewardLabel      string   `json:"volunteer_tasks_filter_reward_label"`
	VolunteerTasksFilterTeamLabel        string   `json:"volunteer_tasks_filter_team_label"`
	VolunteerTasksFilterOnlineLabel      string   `json:"volunteer_tasks_filter_online_label"`
	VolunteerTasksFilterEmptyText        string   `json:"volunteer_tasks_filter_empty_text"`
	VolunteerTasksLocationMissingText    string   `json:"volunteer_tasks_location_missing_text"`
	VolunteerTasksLocationUpdateButton   string   `json:"volunteer_tasks_location_update_button"`
	VolunteerTasksLocationSkipButton     string   `json:"volunteer_tasks_location_skip_button"`
	VolunteerTasksLocationSkipText       string   `json:"volunteer_tasks_location_skip_text"`
	VolunteerTasksLocationUpdatedText    string   `json:"volunteer_tasks_location_updated_text"`
	VolunteerTasksListItemFormat         string   `json:"volunteer_tasks_list_item_format"`
	VolunteerTasksListItemLocation       string   `json:"volunteer_tasks_list_item_location"`
	VolunteerTasksListItemReward         string   `json:"volunteer_tasks_list_item_reward"`
	VolunteerTasksListItemNoReward       string   `json:"volunteer_tasks_list_item_no_reward"`
	VolunteerTasksListItemVolunteersOne  string   `json:"volunteer_tasks_list_item_volunteers_one"`
	VolunteerTasksListItemVolunteersMany string   `json:"volunteer_tasks_list_item_volunteers_many"`
	VolunteerTaskAssignmentsEmptyText    string   `json:"volunteer_task_assignments_empty_text"`
	VolunteerTaskItemTemplate            string   `json:"volunteer_task_item_template"`
	VolunteerOnDemandEmptyText           string   `json:"volunteer_on_demand_empty_text"`
	VolunteerTasksPrevButton             string   `json:"volunteer_tasks_prev_button"`
	VolunteerTasksNextButton             string   `json:"volunteer_tasks_next_button"`
	VolunteerTasksPageFooter             string   `json:"volunteer_tasks_page_footer"`
	VolunteerTaskRewardNotification      string   `json:"volunteer_task_reward_notification"`
	TaskServiceUnavailableText           string   `json:"task_service_unavailable_text"`
	TaskFetchErrorText                   string   `json:"task_fetch_error_text"`
	TaskCreateNoCustomerText             string   `json:"task_create_no_customer_text"`
	TaskCreateNamePrompt                 string   `json:"task_create_name_prompt"`
	TaskCreateNameRetryText              string   `json:"task_create_name_retry_text"`
	TaskCreateDescriptionPrompt          string   `json:"task_create_description_prompt"`
	TaskCreateDescriptionRetryText       string   `json:"task_create_description_retry_text"`
	TaskCreateSuccessText                string   `json:"task_create_success_text"`
	TaskCreateErrorText                  string   `json:"task_create_error_text"`
	TaskCreateFormatPrompt               string   `json:"task_create_format_prompt"`
	TaskCreateFormatOfflineButton        string   `json:"task_create_format_offline_button"`
	TaskCreateFormatOnlineButton         string   `json:"task_create_format_online_button"`
	TaskCreateFormatOfflineLabel         string   `json:"task_create_format_offline_label"`
	TaskCreateFormatOnlineLabel          string   `json:"task_create_format_online_label"`
	TaskCreateLocationPrompt             string   `json:"task_create_location_prompt"`
	TaskCreateLocationRetryText          string   `json:"task_create_location_retry_text"`
	TaskCreateLocationSendButton         string   `json:"task_create_location_send_button"`
	TaskCreateLocationSkipButton         string   `json:"task_create_location_skip_button"`
	TaskCreateLocationFallbackLabel      string   `json:"task_create_location_fallback_label"`
	TaskCreateRewardPrompt               string   `json:"task_create_reward_prompt"`
	TaskCreateRewardRetryText            string   `json:"task_create_reward_retry_text"`
	TaskCreateRewardSkipButton           string   `json:"task_create_reward_skip_button"`
	TaskCreateMembersPrompt              string   `json:"task_create_members_prompt"`
	TaskCreateMembersRetryText           string   `json:"task_create_members_retry_text"`
	TaskCreateMembersSkipButton          string   `json:"task_create_members_skip_button"`
	TaskCreateReviewTemplate             string   `json:"task_create_review_template"`
	TaskCreateReviewConfirmButton        string   `json:"task_create_review_confirm_button"`
	TaskCreateRestartButton              string   `json:"task_create_restart_button"`
	TaskCreateReviewNoReward             string   `json:"task_create_review_no_reward"`
	VolunteerTaskDetailTitle             string   `json:"volunteer_task_detail_title"`
	VolunteerTaskJoinButton              string   `json:"volunteer_task_join_button"`
	VolunteerTaskLeaveButton             string   `json:"volunteer_task_leave_button"`
	VolunteerTaskConfirmButton           string   `json:"volunteer_task_confirm_button"`
	VolunteerTaskJoinSuccessText         string   `json:"volunteer_task_join_success_text"`
	VolunteerTaskJoinErrorText           string   `json:"volunteer_task_join_error_text"`
	VolunteerTaskLeaveSuccessText        string   `json:"volunteer_task_leave_success_text"`
	VolunteerTaskLeaveErrorText          string   `json:"volunteer_task_leave_error_text"`
	VolunteerTaskConfirmSuccessText      string   `json:"volunteer_task_confirm_success_text"`
	VolunteerTaskConfirmErrorText        string   `json:"volunteer_task_confirm_error_text"`
	VolunteerTaskDetailBackButton        string   `json:"volunteer_task_detail_back_button"`
	CustomerTaskDetailTitle              string   `json:"customer_task_detail_title"`
	CustomerTaskApproveButton            string   `json:"customer_task_approve_button"`
	CustomerTaskRejectButton             string   `json:"customer_task_reject_button"`
	CustomerTaskApproveSuccessText       string   `json:"customer_task_approve_success_text"`
	CustomerTaskRejectSuccessText        string   `json:"customer_task_reject_success_text"`
	CustomerTaskDecisionErrorText        string   `json:"customer_task_decision_error_text"`
	CustomerDeleteConfirmText            string   `json:"customer_delete_confirm_text"`
	CustomerDeleteConfirmButton          string   `json:"customer_delete_confirm_button"`
	CustomerDeleteCancelButton           string   `json:"customer_delete_cancel_button"`
	CustomerDeleteSuccessText            string   `json:"customer_delete_success_text"`
	CustomerDeleteErrorText              string   `json:"customer_delete_error_text"`
	ProfileTitle                         string   `json:"profile_title"`
	ProfileSkillsTitle                   string   `json:"profile_skills_title"`
	ProfileLevelBalanceTemplate          string   `json:"profile_level_balance_template"`
	ProfileHistoryButton                 string   `json:"profile_history_button"`
	ProfileEditButton                    string   `json:"profile_edit_button"`
	ProfileSecurityButton                string   `json:"profile_security_button"`
	ProfileBackButton                    string   `json:"profile_back_button"`
	ProfileCoinsButton                   string   `json:"profile_coins_button"`
	ProfileSecurityTitle                 string   `json:"profile_security_title"`
	ProfileSecurityText                  string   `json:"profile_security_text"`
	ProfileSecuritySOSButton             string   `json:"profile_security_sos_button"`
	ProfileSecuritySOSLink               string   `json:"profile_security_sos_link"`
	ProfileHistoryText                   string   `json:"profile_history_text"`
	ProfileEditText                      string   `json:"profile_edit_text"`
	RegistrationStartText                string   `json:"registration_start_text"`
	RegistrationAgeRetryText             string   `json:"registration_age_retry_text"`
	RegistrationAgeUnder18Button         string   `json:"registration_age_under_18_button"`
	RegistrationAge18_24Button           string   `json:"registration_age_18_24_button"`
	RegistrationAge25_34Button           string   `json:"registration_age_25_34_button"`
	RegistrationAge35_44Button           string   `json:"registration_age_35_44_button"`
	RegistrationAge45_54Button           string   `json:"registration_age_45_54_button"`
	RegistrationAge55_64Button           string   `json:"registration_age_55_64_button"`
	RegistrationAge65PlusButton          string   `json:"registration_age_65_plus_button"`
	RegistrationSexPrompt                string   `json:"registration_sex_prompt"`
	RegistrationSexMaleText              string   `json:"registration_sex_male_text"`
	RegistrationSexFemaleText            string   `json:"registration_sex_female_text"`
	RegistrationLocationPrompt           string   `json:"registration_location_prompt"`
	RegistrationLocationGeoButton        string   `json:"registration_location_geo_button"`
	RegistrationLocationSkipButton       string   `json:"registration_location_skip_button"`
	RegistrationLocationRetryText        string   `json:"registration_location_retry_text"`
	RegistrationAboutPrompt              string   `json:"registration_about_prompt"`
	RegistrationAboutConfirmButton       string   `json:"registration_about_confirm_button"`
	RegistrationAboutOptions             []string `json:"registration_about_options"`
	RegistrationErrorText                string   `json:"registration_error_text"`
	RegistrationCompleteText             string   `json:"registration_complete_text"`
	NewUserWelcomeText                   string   `json:"new_user_welcome_text"`
	NewUserJoinButton                    string   `json:"new_user_join_button"`
	CoinsIntroText                       string   `json:"coins_intro_text"`
	CoinsButtons                         []string `json:"coins_buttons"`
	CoinsHowToGetText                    string   `json:"coins_how_to_get_text"`
	CoinsHowToSpendText                  string   `json:"coins_how_to_spend_text"`
	CoinsLevelsText                      string   `json:"coins_levels_text"`
	CoinsBackButton                      string   `json:"coins_back_button"`
	AboutDobrikaText                     string   `json:"about_dobrika_text"`
	AboutDobrikaButtons                  []string `json:"about_dobrika_buttons"`
	AboutDobrikaHowText                  string   `json:"about_dobrika_how_text"`
	AboutDobrikaRulesText                string   `json:"about_dobrika_rules_text"`
	AboutDobrikaInitiatorText            string   `json:"about_dobrika_initiator_text"`
	AboutDobrikaSupportText              string   `json:"about_dobrika_support_text"`
}

var (
	once    sync.Once
	cached  Messages
	loadErr error
)

func Load() (Messages, error) {
	once.Do(func() {
		defaults := defaultMessages()

		var overrides Messages
		if err := json.Unmarshal(rawRU, &overrides); err != nil {
			loadErr = fmt.Errorf("failed to unmarshal ru.json: %w", err)
			cached = defaults
			return
		}

		cached = mergeMessages(defaults, overrides)
	})
	return cached, loadErr
}

func mergeMessages(base, overrides Messages) Messages {
	if overrides.MainMenuText != "" {
		base.MainMenuText = overrides.MainMenuText
	}
	if len(overrides.MainMenuButtons) > 0 {
		base.MainMenuButtons = overrides.MainMenuButtons
	}
	if overrides.CustomerServiceUnavailableText != "" {
		base.CustomerServiceUnavailableText = overrides.CustomerServiceUnavailableText
	}
	if overrides.CustomerLookupErrorText != "" {
		base.CustomerLookupErrorText = overrides.CustomerLookupErrorText
	}
	if overrides.CustomerFormIntroText != "" {
		base.CustomerFormIntroText = overrides.CustomerFormIntroText
	}
	if overrides.CustomerSummaryTitle != "" {
		base.CustomerSummaryTitle = overrides.CustomerSummaryTitle
	}
	if overrides.CustomerSummaryTemplate != "" {
		base.CustomerSummaryTemplate = overrides.CustomerSummaryTemplate
	}
	if overrides.CustomerTypePrompt != "" {
		base.CustomerTypePrompt = overrides.CustomerTypePrompt
	}
	if overrides.CustomerTypeIndividualButton != "" {
		base.CustomerTypeIndividualButton = overrides.CustomerTypeIndividualButton
	}
	if overrides.CustomerTypeBusinessButton != "" {
		base.CustomerTypeBusinessButton = overrides.CustomerTypeBusinessButton
	}
	if overrides.CustomerTypeIndividualLabel != "" {
		base.CustomerTypeIndividualLabel = overrides.CustomerTypeIndividualLabel
	}
	if overrides.CustomerTypeBusinessLabel != "" {
		base.CustomerTypeBusinessLabel = overrides.CustomerTypeBusinessLabel
	}
	if overrides.CustomerNamePrompt != "" {
		base.CustomerNamePrompt = overrides.CustomerNamePrompt
	}
	if overrides.CustomerNameRetryText != "" {
		base.CustomerNameRetryText = overrides.CustomerNameRetryText
	}
	if overrides.CustomerNamePromptIndividual != "" {
		base.CustomerNamePromptIndividual = overrides.CustomerNamePromptIndividual
	}
	if overrides.CustomerNamePromptBusiness != "" {
		base.CustomerNamePromptBusiness = overrides.CustomerNamePromptBusiness
	}
	if overrides.CustomerNameRetryIndividual != "" {
		base.CustomerNameRetryIndividual = overrides.CustomerNameRetryIndividual
	}
	if overrides.CustomerNameRetryBusiness != "" {
		base.CustomerNameRetryBusiness = overrides.CustomerNameRetryBusiness
	}
	if overrides.CustomerAboutPrompt != "" {
		base.CustomerAboutPrompt = overrides.CustomerAboutPrompt
	}
	if overrides.CustomerAboutRetryText != "" {
		base.CustomerAboutRetryText = overrides.CustomerAboutRetryText
	}
	if overrides.CustomerAboutPromptIndividual != "" {
		base.CustomerAboutPromptIndividual = overrides.CustomerAboutPromptIndividual
	}
	if overrides.CustomerAboutPromptBusiness != "" {
		base.CustomerAboutPromptBusiness = overrides.CustomerAboutPromptBusiness
	}
	if overrides.CustomerAboutRetryIndividual != "" {
		base.CustomerAboutRetryIndividual = overrides.CustomerAboutRetryIndividual
	}
	if overrides.CustomerAboutRetryBusiness != "" {
		base.CustomerAboutRetryBusiness = overrides.CustomerAboutRetryBusiness
	}
	if overrides.CustomerCreateSuccessText != "" {
		base.CustomerCreateSuccessText = overrides.CustomerCreateSuccessText
	}
	if overrides.CustomerUpdateSuccessText != "" {
		base.CustomerUpdateSuccessText = overrides.CustomerUpdateSuccessText
	}
	if overrides.CustomerSaveErrorText != "" {
		base.CustomerSaveErrorText = overrides.CustomerSaveErrorText
	}
	if overrides.CustomerManageCreateButton != "" {
		base.CustomerManageCreateButton = overrides.CustomerManageCreateButton
	}
	if overrides.CustomerManageUpdateButton != "" {
		base.CustomerManageUpdateButton = overrides.CustomerManageUpdateButton
	}
	if overrides.CustomerManageDeleteButton != "" {
		base.CustomerManageDeleteButton = overrides.CustomerManageDeleteButton
	}
	if overrides.CustomerManageTasksButton != "" {
		base.CustomerManageTasksButton = overrides.CustomerManageTasksButton
	}
	if overrides.CustomerManageCreateTaskButton != "" {
		base.CustomerManageCreateTaskButton = overrides.CustomerManageCreateTaskButton
	}
	if overrides.CustomerManageBackButton != "" {
		base.CustomerManageBackButton = overrides.CustomerManageBackButton
	}
	if overrides.CustomerDeleteConfirmText != "" {
		base.CustomerDeleteConfirmText = overrides.CustomerDeleteConfirmText
	}
	if overrides.CustomerDeleteConfirmButton != "" {
		base.CustomerDeleteConfirmButton = overrides.CustomerDeleteConfirmButton
	}
	if overrides.CustomerDeleteCancelButton != "" {
		base.CustomerDeleteCancelButton = overrides.CustomerDeleteCancelButton
	}
	if overrides.CustomerTasksListText != "" {
		base.CustomerTasksListText = overrides.CustomerTasksListText
	}
	if overrides.CustomerCreateTaskPlaceholderText != "" {
		base.CustomerCreateTaskPlaceholderText = overrides.CustomerCreateTaskPlaceholderText
	}
	if overrides.CustomerTasksEmptyText != "" {
		base.CustomerTasksEmptyText = overrides.CustomerTasksEmptyText
	}
	if overrides.CustomerTaskItemTemplate != "" {
		base.CustomerTaskItemTemplate = overrides.CustomerTaskItemTemplate
	}
	if overrides.CustomerTasksPrevButton != "" {
		base.CustomerTasksPrevButton = overrides.CustomerTasksPrevButton
	}
	if overrides.CustomerTasksNextButton != "" {
		base.CustomerTasksNextButton = overrides.CustomerTasksNextButton
	}
	if overrides.CustomerTasksPageFooter != "" {
		base.CustomerTasksPageFooter = overrides.CustomerTasksPageFooter
	}
	if overrides.CustomerTaskRewardDescription != "" {
		base.CustomerTaskRewardDescription = overrides.CustomerTaskRewardDescription
	}
	if overrides.CustomerTaskDetailFormat != "" {
		base.CustomerTaskDetailFormat = overrides.CustomerTaskDetailFormat
	}
	if overrides.CustomerTaskDetailLocation != "" {
		base.CustomerTaskDetailLocation = overrides.CustomerTaskDetailLocation
	}
	if overrides.CustomerTaskDetailReward != "" {
		base.CustomerTaskDetailReward = overrides.CustomerTaskDetailReward
	}
	if overrides.CustomerTaskDetailNoReward != "" {
		base.CustomerTaskDetailNoReward = overrides.CustomerTaskDetailNoReward
	}
	if overrides.CustomerTaskDetailVolunteersOne != "" {
		base.CustomerTaskDetailVolunteersOne = overrides.CustomerTaskDetailVolunteersOne
	}
	if overrides.CustomerTaskDetailVolunteersMany != "" {
		base.CustomerTaskDetailVolunteersMany = overrides.CustomerTaskDetailVolunteersMany
	}
	if overrides.CustomerTaskDetailCreatedAt != "" {
		base.CustomerTaskDetailCreatedAt = overrides.CustomerTaskDetailCreatedAt
	}
	if overrides.CustomerTaskAssignmentsEmptyText != "" {
		base.CustomerTaskAssignmentsEmptyText = overrides.CustomerTaskAssignmentsEmptyText
	}
	if overrides.VolunteerMenuIntro != "" {
		base.VolunteerMenuIntro = overrides.VolunteerMenuIntro
	}
	if overrides.VolunteerMenuOnDemandButton != "" {
		base.VolunteerMenuOnDemandButton = overrides.VolunteerMenuOnDemandButton
	}
	if overrides.VolunteerMenuTasksButton != "" {
		base.VolunteerMenuTasksButton = overrides.VolunteerMenuTasksButton
	}
	if overrides.VolunteerMenuProfileButton != "" {
		base.VolunteerMenuProfileButton = overrides.VolunteerMenuProfileButton
	}
	if overrides.VolunteerMenuBackButton != "" {
		base.VolunteerMenuBackButton = overrides.VolunteerMenuBackButton
	}
	if overrides.VolunteerMenuMainButton != "" {
		base.VolunteerMenuMainButton = overrides.VolunteerMenuMainButton
	}
	if overrides.VolunteerOnDemandPlaceholder != "" {
		base.VolunteerOnDemandPlaceholder = overrides.VolunteerOnDemandPlaceholder
	}
	if overrides.VolunteerTasksPlaceholder != "" {
		base.VolunteerTasksPlaceholder = overrides.VolunteerTasksPlaceholder
	}
	if overrides.VolunteerTasksUnavailableText != "" {
		base.VolunteerTasksUnavailableText = overrides.VolunteerTasksUnavailableText
	}
	if overrides.VolunteerTasksErrorText != "" {
		base.VolunteerTasksErrorText = overrides.VolunteerTasksErrorText
	}
	if overrides.VolunteerTasksEmptyText != "" {
		base.VolunteerTasksEmptyText = overrides.VolunteerTasksEmptyText
	}
	if overrides.VolunteerTasksFilterAllButton != "" {
		base.VolunteerTasksFilterAllButton = overrides.VolunteerTasksFilterAllButton
	}
	if overrides.VolunteerTasksFilterRewardButton != "" {
		base.VolunteerTasksFilterRewardButton = overrides.VolunteerTasksFilterRewardButton
	}
	if overrides.VolunteerTasksFilterTeamButton != "" {
		base.VolunteerTasksFilterTeamButton = overrides.VolunteerTasksFilterTeamButton
	}
	if overrides.VolunteerTasksFilterOnlineButton != "" {
		base.VolunteerTasksFilterOnlineButton = overrides.VolunteerTasksFilterOnlineButton
	}
	if overrides.VolunteerTasksFilterAllLabel != "" {
		base.VolunteerTasksFilterAllLabel = overrides.VolunteerTasksFilterAllLabel
	}
	if overrides.VolunteerTasksFilterRewardLabel != "" {
		base.VolunteerTasksFilterRewardLabel = overrides.VolunteerTasksFilterRewardLabel
	}
	if overrides.VolunteerTasksFilterTeamLabel != "" {
		base.VolunteerTasksFilterTeamLabel = overrides.VolunteerTasksFilterTeamLabel
	}
	if overrides.VolunteerTasksFilterOnlineLabel != "" {
		base.VolunteerTasksFilterOnlineLabel = overrides.VolunteerTasksFilterOnlineLabel
	}
	if overrides.VolunteerTasksFilterEmptyText != "" {
		base.VolunteerTasksFilterEmptyText = overrides.VolunteerTasksFilterEmptyText
	}
	if overrides.VolunteerTasksLocationMissingText != "" {
		base.VolunteerTasksLocationMissingText = overrides.VolunteerTasksLocationMissingText
	}
	if overrides.VolunteerTasksLocationUpdateButton != "" {
		base.VolunteerTasksLocationUpdateButton = overrides.VolunteerTasksLocationUpdateButton
	}
	if overrides.VolunteerTasksLocationSkipButton != "" {
		base.VolunteerTasksLocationSkipButton = overrides.VolunteerTasksLocationSkipButton
	}
	if overrides.VolunteerTasksLocationSkipText != "" {
		base.VolunteerTasksLocationSkipText = overrides.VolunteerTasksLocationSkipText
	}
	if overrides.VolunteerTasksLocationUpdatedText != "" {
		base.VolunteerTasksLocationUpdatedText = overrides.VolunteerTasksLocationUpdatedText
	}
	if overrides.VolunteerTasksListItemFormat != "" {
		base.VolunteerTasksListItemFormat = overrides.VolunteerTasksListItemFormat
	}
	if overrides.VolunteerTasksListItemLocation != "" {
		base.VolunteerTasksListItemLocation = overrides.VolunteerTasksListItemLocation
	}
	if overrides.VolunteerTasksListItemReward != "" {
		base.VolunteerTasksListItemReward = overrides.VolunteerTasksListItemReward
	}
	if overrides.VolunteerTasksListItemNoReward != "" {
		base.VolunteerTasksListItemNoReward = overrides.VolunteerTasksListItemNoReward
	}
	if overrides.VolunteerTasksListItemVolunteersOne != "" {
		base.VolunteerTasksListItemVolunteersOne = overrides.VolunteerTasksListItemVolunteersOne
	}
	if overrides.VolunteerTasksListItemVolunteersMany != "" {
		base.VolunteerTasksListItemVolunteersMany = overrides.VolunteerTasksListItemVolunteersMany
	}
	if overrides.VolunteerTaskAssignmentsEmptyText != "" {
		base.VolunteerTaskAssignmentsEmptyText = overrides.VolunteerTaskAssignmentsEmptyText
	}
	if overrides.VolunteerTaskItemTemplate != "" {
		base.VolunteerTaskItemTemplate = overrides.VolunteerTaskItemTemplate
	}
	if overrides.VolunteerOnDemandEmptyText != "" {
		base.VolunteerOnDemandEmptyText = overrides.VolunteerOnDemandEmptyText
	}
	if overrides.VolunteerTasksPrevButton != "" {
		base.VolunteerTasksPrevButton = overrides.VolunteerTasksPrevButton
	}
	if overrides.VolunteerTasksNextButton != "" {
		base.VolunteerTasksNextButton = overrides.VolunteerTasksNextButton
	}
	if overrides.VolunteerTasksPageFooter != "" {
		base.VolunteerTasksPageFooter = overrides.VolunteerTasksPageFooter
	}
	if overrides.VolunteerTaskRewardNotification != "" {
		base.VolunteerTaskRewardNotification = overrides.VolunteerTaskRewardNotification
	}
	if overrides.TaskServiceUnavailableText != "" {
		base.TaskServiceUnavailableText = overrides.TaskServiceUnavailableText
	}
	if overrides.TaskFetchErrorText != "" {
		base.TaskFetchErrorText = overrides.TaskFetchErrorText
	}
	if overrides.TaskCreateNoCustomerText != "" {
		base.TaskCreateNoCustomerText = overrides.TaskCreateNoCustomerText
	}
	if overrides.TaskCreateNamePrompt != "" {
		base.TaskCreateNamePrompt = overrides.TaskCreateNamePrompt
	}
	if overrides.TaskCreateNameRetryText != "" {
		base.TaskCreateNameRetryText = overrides.TaskCreateNameRetryText
	}
	if overrides.TaskCreateDescriptionPrompt != "" {
		base.TaskCreateDescriptionPrompt = overrides.TaskCreateDescriptionPrompt
	}
	if overrides.TaskCreateDescriptionRetryText != "" {
		base.TaskCreateDescriptionRetryText = overrides.TaskCreateDescriptionRetryText
	}
	if overrides.TaskCreateSuccessText != "" {
		base.TaskCreateSuccessText = overrides.TaskCreateSuccessText
	}
	if overrides.TaskCreateErrorText != "" {
		base.TaskCreateErrorText = overrides.TaskCreateErrorText
	}
	if overrides.TaskCreateFormatPrompt != "" {
		base.TaskCreateFormatPrompt = overrides.TaskCreateFormatPrompt
	}
	if overrides.TaskCreateFormatOfflineButton != "" {
		base.TaskCreateFormatOfflineButton = overrides.TaskCreateFormatOfflineButton
	}
	if overrides.TaskCreateFormatOnlineButton != "" {
		base.TaskCreateFormatOnlineButton = overrides.TaskCreateFormatOnlineButton
	}
	if overrides.TaskCreateFormatOfflineLabel != "" {
		base.TaskCreateFormatOfflineLabel = overrides.TaskCreateFormatOfflineLabel
	}
	if overrides.TaskCreateFormatOnlineLabel != "" {
		base.TaskCreateFormatOnlineLabel = overrides.TaskCreateFormatOnlineLabel
	}
	if overrides.TaskCreateLocationPrompt != "" {
		base.TaskCreateLocationPrompt = overrides.TaskCreateLocationPrompt
	}
	if overrides.TaskCreateLocationRetryText != "" {
		base.TaskCreateLocationRetryText = overrides.TaskCreateLocationRetryText
	}
	if overrides.TaskCreateLocationSendButton != "" {
		base.TaskCreateLocationSendButton = overrides.TaskCreateLocationSendButton
	}
	if overrides.TaskCreateLocationSkipButton != "" {
		base.TaskCreateLocationSkipButton = overrides.TaskCreateLocationSkipButton
	}
	if overrides.TaskCreateLocationFallbackLabel != "" {
		base.TaskCreateLocationFallbackLabel = overrides.TaskCreateLocationFallbackLabel
	}
	if overrides.TaskCreateRewardPrompt != "" {
		base.TaskCreateRewardPrompt = overrides.TaskCreateRewardPrompt
	}
	if overrides.TaskCreateRewardRetryText != "" {
		base.TaskCreateRewardRetryText = overrides.TaskCreateRewardRetryText
	}
	if overrides.TaskCreateRewardSkipButton != "" {
		base.TaskCreateRewardSkipButton = overrides.TaskCreateRewardSkipButton
	}
	if overrides.TaskCreateMembersPrompt != "" {
		base.TaskCreateMembersPrompt = overrides.TaskCreateMembersPrompt
	}
	if overrides.TaskCreateMembersRetryText != "" {
		base.TaskCreateMembersRetryText = overrides.TaskCreateMembersRetryText
	}
	if overrides.TaskCreateMembersSkipButton != "" {
		base.TaskCreateMembersSkipButton = overrides.TaskCreateMembersSkipButton
	}
	if overrides.TaskCreateReviewTemplate != "" {
		base.TaskCreateReviewTemplate = overrides.TaskCreateReviewTemplate
	}
	if overrides.TaskCreateReviewConfirmButton != "" {
		base.TaskCreateReviewConfirmButton = overrides.TaskCreateReviewConfirmButton
	}
	if overrides.TaskCreateRestartButton != "" {
		base.TaskCreateRestartButton = overrides.TaskCreateRestartButton
	}
	if overrides.TaskCreateReviewNoReward != "" {
		base.TaskCreateReviewNoReward = overrides.TaskCreateReviewNoReward
	}
	if overrides.VolunteerTaskDetailTitle != "" {
		base.VolunteerTaskDetailTitle = overrides.VolunteerTaskDetailTitle
	}
	if overrides.VolunteerTaskJoinButton != "" {
		base.VolunteerTaskJoinButton = overrides.VolunteerTaskJoinButton
	}
	if overrides.VolunteerTaskLeaveButton != "" {
		base.VolunteerTaskLeaveButton = overrides.VolunteerTaskLeaveButton
	}
	if overrides.VolunteerTaskConfirmButton != "" {
		base.VolunteerTaskConfirmButton = overrides.VolunteerTaskConfirmButton
	}
	if overrides.VolunteerTaskJoinSuccessText != "" {
		base.VolunteerTaskJoinSuccessText = overrides.VolunteerTaskJoinSuccessText
	}
	if overrides.VolunteerTaskJoinErrorText != "" {
		base.VolunteerTaskJoinErrorText = overrides.VolunteerTaskJoinErrorText
	}
	if overrides.VolunteerTaskLeaveSuccessText != "" {
		base.VolunteerTaskLeaveSuccessText = overrides.VolunteerTaskLeaveSuccessText
	}
	if overrides.VolunteerTaskLeaveErrorText != "" {
		base.VolunteerTaskLeaveErrorText = overrides.VolunteerTaskLeaveErrorText
	}
	if overrides.VolunteerTaskConfirmSuccessText != "" {
		base.VolunteerTaskConfirmSuccessText = overrides.VolunteerTaskConfirmSuccessText
	}
	if overrides.VolunteerTaskConfirmErrorText != "" {
		base.VolunteerTaskConfirmErrorText = overrides.VolunteerTaskConfirmErrorText
	}
	if overrides.VolunteerTaskDetailBackButton != "" {
		base.VolunteerTaskDetailBackButton = overrides.VolunteerTaskDetailBackButton
	}
	if overrides.CustomerTaskDetailTitle != "" {
		base.CustomerTaskDetailTitle = overrides.CustomerTaskDetailTitle
	}
	if overrides.CustomerTaskApproveButton != "" {
		base.CustomerTaskApproveButton = overrides.CustomerTaskApproveButton
	}
	if overrides.CustomerTaskRejectButton != "" {
		base.CustomerTaskRejectButton = overrides.CustomerTaskRejectButton
	}
	if overrides.CustomerTaskApproveSuccessText != "" {
		base.CustomerTaskApproveSuccessText = overrides.CustomerTaskApproveSuccessText
	}
	if overrides.CustomerTaskRejectSuccessText != "" {
		base.CustomerTaskRejectSuccessText = overrides.CustomerTaskRejectSuccessText
	}
	if overrides.CustomerTaskDecisionErrorText != "" {
		base.CustomerTaskDecisionErrorText = overrides.CustomerTaskDecisionErrorText
	}
	if overrides.CustomerDeleteSuccessText != "" {
		base.CustomerDeleteSuccessText = overrides.CustomerDeleteSuccessText
	}
	if overrides.CustomerDeleteErrorText != "" {
		base.CustomerDeleteErrorText = overrides.CustomerDeleteErrorText
	}
	if overrides.ProfileTitle != "" {
		base.ProfileTitle = overrides.ProfileTitle
	}
	if overrides.ProfileSkillsTitle != "" {
		base.ProfileSkillsTitle = overrides.ProfileSkillsTitle
	}
	if overrides.ProfileLevelBalanceTemplate != "" {
		base.ProfileLevelBalanceTemplate = overrides.ProfileLevelBalanceTemplate
	}
	if overrides.ProfileHistoryButton != "" {
		base.ProfileHistoryButton = overrides.ProfileHistoryButton
	}
	if overrides.ProfileEditButton != "" {
		base.ProfileEditButton = overrides.ProfileEditButton
	}
	if overrides.ProfileSecurityButton != "" {
		base.ProfileSecurityButton = overrides.ProfileSecurityButton
	}
	if overrides.ProfileBackButton != "" {
		base.ProfileBackButton = overrides.ProfileBackButton
	}
	if overrides.ProfileCoinsButton != "" {
		base.ProfileCoinsButton = overrides.ProfileCoinsButton
	}
	if overrides.ProfileHistoryText != "" {
		base.ProfileHistoryText = overrides.ProfileHistoryText
	}
	if overrides.ProfileEditText != "" {
		base.ProfileEditText = overrides.ProfileEditText
	}
	if overrides.ProfileSecurityTitle != "" {
		base.ProfileSecurityTitle = overrides.ProfileSecurityTitle
	}
	if overrides.ProfileSecurityText != "" {
		base.ProfileSecurityText = overrides.ProfileSecurityText
	}
	if overrides.ProfileSecuritySOSButton != "" {
		base.ProfileSecuritySOSButton = overrides.ProfileSecuritySOSButton
	}
	if overrides.ProfileSecuritySOSLink != "" {
		base.ProfileSecuritySOSLink = overrides.ProfileSecuritySOSLink
	}
	if overrides.RegistrationStartText != "" {
		base.RegistrationStartText = overrides.RegistrationStartText
	}
	if overrides.RegistrationAgeRetryText != "" {
		base.RegistrationAgeRetryText = overrides.RegistrationAgeRetryText
	}
	if overrides.RegistrationAgeUnder18Button != "" {
		base.RegistrationAgeUnder18Button = overrides.RegistrationAgeUnder18Button
	}
	if overrides.RegistrationAge18_24Button != "" {
		base.RegistrationAge18_24Button = overrides.RegistrationAge18_24Button
	}
	if overrides.RegistrationAge25_34Button != "" {
		base.RegistrationAge25_34Button = overrides.RegistrationAge25_34Button
	}
	if overrides.RegistrationAge35_44Button != "" {
		base.RegistrationAge35_44Button = overrides.RegistrationAge35_44Button
	}
	if overrides.RegistrationAge45_54Button != "" {
		base.RegistrationAge45_54Button = overrides.RegistrationAge45_54Button
	}
	if overrides.RegistrationAge55_64Button != "" {
		base.RegistrationAge55_64Button = overrides.RegistrationAge55_64Button
	}
	if overrides.RegistrationAge65PlusButton != "" {
		base.RegistrationAge65PlusButton = overrides.RegistrationAge65PlusButton
	}
	if overrides.RegistrationSexPrompt != "" {
		base.RegistrationSexPrompt = overrides.RegistrationSexPrompt
	}
	if overrides.RegistrationSexMaleText != "" {
		base.RegistrationSexMaleText = overrides.RegistrationSexMaleText
	}
	if overrides.RegistrationSexFemaleText != "" {
		base.RegistrationSexFemaleText = overrides.RegistrationSexFemaleText
	}
	if overrides.RegistrationLocationPrompt != "" {
		base.RegistrationLocationPrompt = overrides.RegistrationLocationPrompt
	}
	if overrides.RegistrationLocationGeoButton != "" {
		base.RegistrationLocationGeoButton = overrides.RegistrationLocationGeoButton
	}
	if overrides.RegistrationLocationSkipButton != "" {
		base.RegistrationLocationSkipButton = overrides.RegistrationLocationSkipButton
	}
	if overrides.RegistrationLocationRetryText != "" {
		base.RegistrationLocationRetryText = overrides.RegistrationLocationRetryText
	}
	if overrides.RegistrationAboutPrompt != "" {
		base.RegistrationAboutPrompt = overrides.RegistrationAboutPrompt
	}
	if overrides.RegistrationAboutConfirmButton != "" {
		base.RegistrationAboutConfirmButton = overrides.RegistrationAboutConfirmButton
	}
	if len(overrides.RegistrationAboutOptions) > 0 {
		base.RegistrationAboutOptions = overrides.RegistrationAboutOptions
	}
	if overrides.RegistrationErrorText != "" {
		base.RegistrationErrorText = overrides.RegistrationErrorText
	}
	if overrides.RegistrationCompleteText != "" {
		base.RegistrationCompleteText = overrides.RegistrationCompleteText
	}
	if overrides.NewUserWelcomeText != "" {
		base.NewUserWelcomeText = overrides.NewUserWelcomeText
	}
	if overrides.NewUserJoinButton != "" {
		base.NewUserJoinButton = overrides.NewUserJoinButton
	}
	if overrides.CoinsIntroText != "" {
		base.CoinsIntroText = overrides.CoinsIntroText
	}
	if len(overrides.CoinsButtons) > 0 {
		base.CoinsButtons = overrides.CoinsButtons
	}
	if overrides.CoinsHowToGetText != "" {
		base.CoinsHowToGetText = overrides.CoinsHowToGetText
	}
	if overrides.CoinsHowToSpendText != "" {
		base.CoinsHowToSpendText = overrides.CoinsHowToSpendText
	}
	if overrides.CoinsLevelsText != "" {
		base.CoinsLevelsText = overrides.CoinsLevelsText
	}
	if overrides.CoinsBackButton != "" {
		base.CoinsBackButton = overrides.CoinsBackButton
	}
	if overrides.AboutDobrikaText != "" {
		base.AboutDobrikaText = overrides.AboutDobrikaText
	}
	if len(overrides.AboutDobrikaButtons) > 0 {
		base.AboutDobrikaButtons = overrides.AboutDobrikaButtons
	}
	if overrides.AboutDobrikaHowText != "" {
		base.AboutDobrikaHowText = overrides.AboutDobrikaHowText
	}
	if overrides.AboutDobrikaRulesText != "" {
		base.AboutDobrikaRulesText = overrides.AboutDobrikaRulesText
	}
	if overrides.AboutDobrikaInitiatorText != "" {
		base.AboutDobrikaInitiatorText = overrides.AboutDobrikaInitiatorText
	}
	if overrides.AboutDobrikaSupportText != "" {
		base.AboutDobrikaSupportText = overrides.AboutDobrikaSupportText
	}

	return base
}

func defaultMessages() Messages {
	return Messages{
		MainMenuText: "Главное меню. Что хотите сделать?",
		MainMenuButtons: []string{
			"Хочу помочь",
			"Мне нужна помощь",
			"Мой профиль",
			"О Добрике",
		},
		CustomerServiceUnavailableText:       "Сервис заказчиков недоступен. Попробуйте позже.",
		CustomerLookupErrorText:              "Не удалось получить данные заказчика. Попробуйте позже.",
		CustomerFormIntroText:                "Расскажите о заказчике. Заполните профиль, чтобы волонтёры быстрее откликнулись.",
		CustomerSummaryTitle:                 "Профиль заказчика:",
		CustomerSummaryTemplate:              "*Кому:* %s\n*История:* %s",
		CustomerTypePrompt:                   "Кто обращается за помощью?",
		CustomerTypeIndividualButton:         "Частное лицо",
		CustomerTypeBusinessButton:           "Организация",
		CustomerTypeIndividualLabel:          "Частное лицо",
		CustomerTypeBusinessLabel:            "Организация",
		CustomerNamePrompt:                   "Как вас зовут или как называется организация?",
		CustomerNamePromptIndividual:         "Как вас зовут?",
		CustomerNamePromptBusiness:           "Как называется ваша организация или фонд?",
		CustomerNameRetryText:                "Пожалуйста, укажите имя или название.",
		CustomerNameRetryIndividual:          "Пожалуйста, укажите имя.",
		CustomerNameRetryBusiness:            "Пожалуйста, укажите название организации.",
		CustomerAboutPrompt:                  "Опишите, какая помощь нужна.",
		CustomerAboutPromptIndividual:        "Опишите, какая помощь нужна лично вам или близкому.",
		CustomerAboutPromptBusiness:          "Опишите, какая помощь нужна вашей организации.",
		CustomerAboutRetryText:               "Пожалуйста, опишите, какая помощь нужна.",
		CustomerAboutRetryIndividual:         "Пожалуйста, опишите, какая помощь нужна.",
		CustomerAboutRetryBusiness:           "Пожалуйста, опишите, какая помощь нужна организации.",
		CustomerCreateSuccessText:            "Спасибо! Профиль заказчика сохранён.",
		CustomerUpdateSuccessText:            "Профиль заказчика обновлён.",
		CustomerSaveErrorText:                "Не удалось сохранить профиль. Попробуйте позже.",
		CustomerManageCreateButton:           "Заполнить профиль",
		CustomerManageUpdateButton:           "Обновить профиль",
		CustomerManageDeleteButton:           "Удалить профиль",
		CustomerManageBackButton:             "⬅️ Назад в меню",
		CustomerManageTasksButton:            "Мои задачи",
		CustomerManageCreateTaskButton:       "Создать задачу",
		CustomerTasksListText:                "Список добрых дел:",
		CustomerCreateTaskPlaceholderText:    "Создание задач появится позже. Следите за обновлениями!",
		CustomerTasksEmptyText:               "Пока задач нет. Создайте первое доброе дело!",
		CustomerTaskItemTemplate:             "• *%s*\n%s",
		CustomerTasksPrevButton:              "⬅️ Назад",
		CustomerTasksNextButton:              "➡️ Далее",
		CustomerTasksPageFooter:              "Страница %d из %d",
		CustomerTaskRewardDescription:        "Награда за выполнение задачи «%s»",
		CustomerTaskDetailFormat:             "Формат: %s",
		CustomerTaskDetailLocation:           "Локация: %s",
		CustomerTaskDetailReward:             "Награда: %d добриков",
		CustomerTaskDetailNoReward:           "Награда: не предусмотрена",
		CustomerTaskDetailVolunteersOne:      "Нужен 1 волонтёр",
		CustomerTaskDetailVolunteersMany:     "Нужно волонтёров: %d",
		CustomerTaskDetailCreatedAt:          "Создано: %s",
		CustomerTaskAssignmentsEmptyText:     "Пока нет откликов на это доброе дело.",
		VolunteerMenuIntro:                   "💚 Выберите, как хотите помочь:",
		VolunteerMenuOnDemandButton:          "По запросу",
		VolunteerMenuTasksButton:             "Список дел",
		VolunteerMenuProfileButton:           "Мой профиль",
		VolunteerMenuBackButton:              "Назад",
		VolunteerMenuMainButton:              "Главное меню",
		VolunteerOnDemandPlaceholder:         "Раздел «По запросу» в разработке. Скоро здесь появятся обращения от людей рядом 💚",
		VolunteerTasksPlaceholder:            "Список дел появится скоро. Здесь будут доступные добрые дела.",
		VolunteerTasksUnavailableText:        "Сервис задач недоступен. Попробуйте позже.",
		VolunteerTasksErrorText:              "Не удалось получить список добрых дел. Попробуйте позже.",
		VolunteerTasksEmptyText:              "Сейчас нет активных задач. Загляните позже!",
		VolunteerTasksFilterAllButton:        "📍 Рядом",
		VolunteerTasksFilterRewardButton:     "💰 Награда",
		VolunteerTasksFilterTeamButton:       "👥 Команда",
		VolunteerTasksFilterOnlineButton:     "💻 Онлайн",
		VolunteerTasksFilterAllLabel:         "все рядом",
		VolunteerTasksFilterRewardLabel:      "с наградой",
		VolunteerTasksFilterTeamLabel:        "для команды",
		VolunteerTasksFilterOnlineLabel:      "онлайн",
		VolunteerTasksFilterEmptyText:        "По фильтру «%s» пока ничего нет. Попробуй другой вариант 💚",
		VolunteerTasksLocationMissingText:    "📍 Отправь локацию кнопкой ниже или пропусти шаг — покажу дела без геопривязки.",
		VolunteerTasksLocationUpdateButton:   "📍 Отправить локацию",
		VolunteerTasksLocationSkipButton:     "Пропустить",
		VolunteerTasksLocationSkipText:       "Показываю доступные дела без учёта геолокации. Если решишь поделиться точкой — просто отправь её кнопкой 💚",
		VolunteerTasksLocationUpdatedText:    "Локация обновлена 💚",
		VolunteerTasksListItemFormat:         "Формат: %s",
		VolunteerTasksListItemLocation:       "Локация: %s",
		VolunteerTasksListItemReward:         "Награда: %d добриков",
		VolunteerTasksListItemNoReward:       "Награда: не предусмотрена",
		VolunteerTasksListItemVolunteersOne:  "Нужен 1 волонтёр",
		VolunteerTasksListItemVolunteersMany: "Нужно волонтёров: %d",
		VolunteerTaskAssignmentsEmptyText:    "Пока никто не откликнулся. Будь первым волонтёром 💚",
		VolunteerTaskItemTemplate:            "• *%s*\n%s",
		VolunteerOnDemandEmptyText:           "У тебя пока нет активных откликов.",
		VolunteerTasksPrevButton:             "⬅️ Назад",
		VolunteerTasksNextButton:             "➡️ Далее",
		VolunteerTasksPageFooter:             "Страница %d из %d",
		VolunteerTaskRewardNotification:      "Спасибо за доброе дело «%s»! Тебе начислено %d добриков 💚",
		TaskServiceUnavailableText:           "Сервис задач недоступен. Попробуйте позже.",
		TaskFetchErrorText:                   "Не удалось получить список задач. Попробуйте позже.",
		TaskCreateNoCustomerText:             "Сначала заполни профиль заказчика, чтобы создавать добрые дела.",
		TaskCreateNamePrompt:                 "Как назовём доброе дело?",
		TaskCreateNameRetryText:              "Введите название доброго дела, пожалуйста.",
		TaskCreateDescriptionPrompt:          "Расскажите, что нужно сделать. Это поможет волонтёрам понять задачу.",
		TaskCreateDescriptionRetryText:       "Добавьте описание, чтобы волонтёры понимали, чем помочь.",
		TaskCreateSuccessText:                "Доброе дело «%s» создано 💚",
		TaskCreateErrorText:                  "Не удалось создать задачу. Попробуйте позже.",
		TaskCreateFormatPrompt:               "Какое это доброе дело? Выберите формат.",
		TaskCreateFormatOfflineButton:        "🏠 Нужно прийти",
		TaskCreateFormatOnlineButton:         "💻 Можно онлайн",
		TaskCreateFormatOfflineLabel:         "офлайн",
		TaskCreateFormatOnlineLabel:          "онлайн",
		TaskCreateLocationPrompt:             "Поделитесь точкой на карте или напишите адрес, где нужна помощь.",
		TaskCreateLocationRetryText:          "Не удалось получить локацию. Попробуйте ещё раз или воспользуйтесь кнопкой отправки геопозиции.",
		TaskCreateLocationSendButton:         "📍 Отправить локацию",
		TaskCreateLocationSkipButton:         "Пропустить локацию",
		TaskCreateLocationFallbackLabel:      "точка на карте",
		TaskCreateRewardPrompt:               "Есть ли награда в добриках? Введите число или нажмите «Без награды».",
		TaskCreateRewardRetryText:            "Нужно указать число. Пример: 50",
		TaskCreateRewardSkipButton:           "Без награды",
		TaskCreateMembersPrompt:              "Сколько волонтёров нужно? Введите число или оставьте 1.",
		TaskCreateMembersRetryText:           "Пожалуйста, укажите число волонтёров (например, 1 или 3).",
		TaskCreateMembersSkipButton:          "Только один",
		TaskCreateReviewTemplate:             "*Проверь детали:*\n\n• Название: %s\n• Описание: %s\n• Формат: %s\n• Локация: %s\n• Награда: %s\n• Волонтёров нужно: %s",
		TaskCreateReviewConfirmButton:        "✅ Опубликовать",
		TaskCreateRestartButton:              "🔄 Заполнить заново",
		TaskCreateReviewNoReward:             "без награды",
		VolunteerTaskDetailTitle:             "*%s*",
		VolunteerTaskJoinButton:              "Откликнуться",
		VolunteerTaskLeaveButton:             "Отказаться",
		VolunteerTaskConfirmButton:           "Я помог(ла)",
		VolunteerTaskJoinSuccessText:         "Отлично! Ты откликнулся(ась) на доброе дело 💚",
		VolunteerTaskJoinErrorText:           "Не получилось откликнуться. Попробуй позже.",
		VolunteerTaskLeaveSuccessText:        "Ты отказался(ась) от участия. Ничего страшного!",
		VolunteerTaskLeaveErrorText:          "Не удалось отказаться от участия. Попробуй позже.",
		VolunteerTaskConfirmSuccessText:      "Спасибо! Мы передали, что ты завершил(а) доброе дело.",
		VolunteerTaskConfirmErrorText:        "Не удалось подтвердить выполнение. Попробуй позже.",
		VolunteerTaskDetailBackButton:        "⬅️ К списку дел",
		CustomerTaskDetailTitle:              "*%s*",
		CustomerTaskApproveButton:            "Подтвердить выполнение",
		CustomerTaskRejectButton:             "Отклонить",
		CustomerTaskApproveSuccessText:       "Выполнение задачи подтверждено 💚",
		CustomerTaskRejectSuccessText:        "Задача помечена как невыполненная.",
		CustomerTaskDecisionErrorText:        "Не удалось обновить статус задачи. Попробуйте позже.",
		CustomerDeleteConfirmText:            "Удалить профиль заказчика?",
		CustomerDeleteConfirmButton:          "Удалить профиль",
		CustomerDeleteCancelButton:           "Отмена",
		CustomerDeleteSuccessText:            "Профиль заказчика удалён.",
		CustomerDeleteErrorText:              "Не удалось удалить профиль. Попробуйте позже.",
		ProfileTitle:                         "👤 *Мой профиль*",
		ProfileSkillsTitle:                   "Навыки и интересы:",
		ProfileLevelBalanceTemplate:          "🎖 Уровень: *%s*\n💰 Репутация: *%d* добриков",
		ProfileHistoryButton:                 "📜 История дел",
		ProfileEditButton:                    "✏️ Редактировать",
		ProfileSecurityButton:                "🛡 Безопасность",
		ProfileBackButton:                    "⬅️ Назад в меню",
		ProfileCoinsButton:                   "💰 Добрики",
		ProfileHistoryText:                   "История добрых дел появится совсем скоро 💚",
		ProfileEditText:                      "Редактирование профиля появится в ближайшем обновлении.",
		ProfileSecurityTitle:                 "🛡 Безопасность встреч офлайн",
		ProfileSecurityText:                  "• Назначайте встречи только в людных местах\n• Делитесь планами с близкими\n• Пользуйтесь кнопкой SOS в экстренных ситуациях\n\nВсе правила и контакты: %s",
		ProfileSecuritySOSButton:             "🚨 Открыть памятку",
		ProfileSecuritySOSLink:               "https://dobrika.example/safety",
		AboutDobrikaText:                     "Добрика — бот добрых дел. Здесь можно помогать другим и получать добрики за сделанное добро.",
		AboutDobrikaButtons: []string{
			"💚 Как это работает",
			"🧭 Правила и безопасность",
			"🏢 Стать инициатором",
			"📞 Связаться с поддержкой",
			"⬅️ Назад в меню",
		},
		AboutDobrikaHowText:            "1. Выбери доброе дело рядом или онлайн.\n2. Выполни его и отправь подтверждение.\n3. Получи добрики и расти в Добрике!",
		AboutDobrikaRulesText:          "Следуй простым правилам безопасности: назначай встречи в людных местах, предупреждай родных и бери с собой телефон.",
		AboutDobrikaInitiatorText:      "Хочешь разместить доброе дело или инициативу? Напиши нам — поможем организовать и привлечь волонтёров.",
		AboutDobrikaSupportText:        "Служба поддержки Добрики отвечает в рабочее время. Пиши на support@dobrika.example или в чат @dobrika_support.",
		RegistrationStartText:          "🎂 Укажите ваш возраст:",
		RegistrationAgeRetryText:       "Выберите вариант на клавиатуре или укажите число.",
		RegistrationAgeUnder18Button:   "< 18 лет",
		RegistrationAge18_24Button:     "18–24 года",
		RegistrationAge25_34Button:     "25–34 года",
		RegistrationAge35_44Button:     "35–44 года",
		RegistrationAge45_54Button:     "45–54 года",
		RegistrationAge55_64Button:     "55–64 года",
		RegistrationAge65PlusButton:    "65+ лет",
		RegistrationSexPrompt:          "Выберите пол:",
		RegistrationSexMaleText:        "Мужчина",
		RegistrationSexFemaleText:      "Женщина",
		RegistrationLocationPrompt:     "Где вы сейчас находитесь? Можно отправить геопозицию или ответить текстом.",
		RegistrationLocationGeoButton:  "Отправить геолокацию",
		RegistrationLocationSkipButton: "Не хочу делиться",
		RegistrationLocationRetryText:  "Не смог получить локацию. Попробуйте ещё раз или отправьте название города текстом.",
		RegistrationAboutPrompt:        "Расскажите, как вы готовы помогать",
		RegistrationAboutConfirmButton: "Подтвердить выбор",
		RegistrationAboutOptions: []string{
			"🛒 Магазин",
			"💬 Разговор",
			"👩‍💻 Помогу удаленно",
			"📦 Доставить",
			"📚 Учеба",
			"🧹 Помогу по дому",
			"🚗 Подвезти",
			"💰 Деньгами",
			"🐾 Питомцы",
			"🤷‍♂️ Не знаю",
		},
		RegistrationErrorText:    "Не удалось сохранить данные. Попробуйте позже.",
		RegistrationCompleteText: "Спасибо! Мы сохранили ваши данные.",
		NewUserWelcomeText:       "Добро пожаловать! Нажмите, чтобы присоединиться.",
		NewUserJoinButton:        "Присоединиться",
		CoinsIntroText:           "Добрики — благодарность за твою помощь. Чем больше добрых дел, тем больше добриков и выше уровень.",
		CoinsButtons: []string{
			"Как получить",
			"На что потратить",
			"Уровни",
			"⬅️ Назад в профиль",
		},
		CoinsHowToGetText:   "Получай добрики, выполняя задания, помогая людям и подтверждая добрые дела.",
		CoinsHowToSpendText: "Добрики можно обменять на сувениры, участвовать в челленджах и дарить друзьям.",
		CoinsLevelsText:     "Каждый уровень открывает новые задания и показывает твою активность в сообществе.",
		CoinsBackButton:     "⬅️ Назад в профиль",
	}
}
