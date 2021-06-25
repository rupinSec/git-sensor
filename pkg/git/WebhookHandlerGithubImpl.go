/*
 * Copyright (c) 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package git

import (
	"github.com/devtron-labs/git-sensor/internal/sql"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"time"
)

type WebhookHandlerGithubImpl struct {
	logger *zap.SugaredLogger
	webhookEventService WebhookEventService
}

func NewWebhookHandlerGithubImpl(logger *zap.SugaredLogger, webhookEventService WebhookEventService) *WebhookHandlerGithubImpl {
	return &WebhookHandlerGithubImpl{
		logger: logger,
		webhookEventService: webhookEventService,
	}
}

func (impl WebhookHandlerGithubImpl) HandleWebhookEvent(requestPayloadJson string) error{

	gitHostName := GIT_HOST_NAME_GITHUB

	// store in webhook_event_json table
	webhookEventJson := &sql.WebhookEventJson{
		GitHostName: gitHostName,
		PayloadJson: requestPayloadJson,
		CreatedOn: time.Now(),
	}

	err := impl.webhookEventService.SaveWebhookEventJson(webhookEventJson)
	if err != nil{
		impl.logger.Errorw("error in saving webhook event json in db","err", err)
		return err
	}


	// parse payload
	prId := gjson.Get(requestPayloadJson, "pull_request.id").String()
	prUrl := gjson.Get(requestPayloadJson, "pull_request.html_url").String()
	prTitle := gjson.Get(requestPayloadJson, "pull_request.title").String()
	sourceBranchName := gjson.Get(requestPayloadJson, "pull_request.head.ref").String()
	sourceBranchHash := gjson.Get(requestPayloadJson, "pull_request.head.sha").String()
	targetBranchName := gjson.Get(requestPayloadJson, "pull_request.base.ref").String()
	targetBranchHash := gjson.Get(requestPayloadJson, "pull_request.base.sha").String()
	prCreatedOn := gjson.Get(requestPayloadJson, "pull_request.created_at").Time()
	prUpdatedOn := gjson.Get(requestPayloadJson, "pull_request.updated_at").Time()
	isOpen := gjson.Get(requestPayloadJson, "pull_request.state").String() == "open"
	currentState := gjson.Get(requestPayloadJson, "action").String()
	repositoryUrl := gjson.Get(requestPayloadJson, "repository.html_url").String()
	authorName := gjson.Get(requestPayloadJson, "sender.login").String()

	// get current pull request data from DB
	webhookEventGetData, err := impl.webhookEventService.GetWebhookPrEventDataByGitHostNameAndPrId(gitHostName, prId)

	// add/update latest information
	webhookEventSaveData := &sql.WebhookPRDataEvent{
		GitHostName: gitHostName,
		PrId: prId,
		PrUrl: prUrl,
		PrTitle: prTitle,
		SourceBranchName: sourceBranchName,
		SourceBranchHash: sourceBranchHash,
		TargetBranchName: targetBranchName,
		TargetBranchHash: targetBranchHash,
		PrCreatedOn: prCreatedOn,
		PrUpdatedOn: prUpdatedOn,
		IsOpen: isOpen,
		ActualState: currentState,
		RepositoryUrl: repositoryUrl,
		AuthorName: authorName,
	}

	if webhookEventGetData != nil {
		webhookEventSaveData.Id = webhookEventGetData.Id
		webhookEventSaveData.CreatedOn = webhookEventGetData.CreatedOn
		webhookEventSaveData.UpdatedOn = time.Now()
		impl.webhookEventService.UpdateWebhookPrEventData(webhookEventSaveData)
	}else{
		webhookEventSaveData.CreatedOn = time.Now()
		impl.webhookEventService.SaveWebhookPrEventData(webhookEventSaveData)
	}


	err = impl.webhookEventService.HandlePostSavePrWebhook(webhookEventSaveData)
	if err != nil{
		impl.logger.Errorw("error in handling pr webhook after db save","err", err)
		return err
	}

	return nil
}