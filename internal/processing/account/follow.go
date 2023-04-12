// GoToSocial
// Copyright (C) GoToSocial Authors admin@gotosocial.org
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/superseriousbusiness/gotosocial/internal/ap"
	apimodel "github.com/superseriousbusiness/gotosocial/internal/api/model"
	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtscontext"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/id"
	"github.com/superseriousbusiness/gotosocial/internal/messages"
	"github.com/superseriousbusiness/gotosocial/internal/uris"
)

// FollowCreate handles a follow request to an account, either remote or local.
func (p *Processor) FollowCreate(ctx context.Context, requestingAccount *gtsmodel.Account, form *apimodel.AccountFollowRequest) (*apimodel.Relationship, gtserror.WithCode) {
	targetAccount, errWithCode := p.getFollowTarget(ctx, requestingAccount.ID, form.ID)
	if errWithCode != nil {
		return nil, errWithCode
	}

	// Check if a follow exists already.
	if follow, err := p.state.DB.GetFollow(
		gtscontext.SetBarebones(ctx),
		requestingAccount.ID,
		targetAccount.ID,
	); err != nil && !errors.Is(err, db.ErrNoEntries) {
		err = fmt.Errorf("FollowCreate: db error checking existing follow: %w", err)
		return nil, gtserror.NewErrorInternalError(err)
	} else if follow != nil {
		// Already follows, update if necessary + return relationship.
		return p.updateFollow(
			ctx,
			requestingAccount,
			form,
			follow.ShowReblogs,
			follow.Notify,
			func(columns ...string) error { return p.state.DB.UpdateFollow(ctx, follow, columns...) },
		)
	}

	// Check if a follow request exists already.
	if followRequest, err := p.state.DB.GetFollowRequest(
		gtscontext.SetBarebones(ctx),
		requestingAccount.ID,
		targetAccount.ID,
	); err != nil && !errors.Is(err, db.ErrNoEntries) {
		err = fmt.Errorf("FollowCreate: db error checking existing follow request: %w", err)
		return nil, gtserror.NewErrorInternalError(err)
	} else if followRequest != nil {
		// Already requested, update if necessary + return relationship.
		return p.updateFollow(
			ctx,
			requestingAccount,
			form,
			followRequest.ShowReblogs,
			followRequest.Notify,
			func(columns ...string) error { return p.state.DB.UpdateFollowRequest(ctx, followRequest, columns...) },
		)
	}

	// Neither follows nor follow requests, so
	// create and store a new follow request.
	followID, err := id.NewRandomULID()
	if err != nil {
		return nil, gtserror.NewErrorInternalError(err)
	}
	followURI := uris.GenerateURIForFollow(requestingAccount.Username, followID)

	fr := &gtsmodel.FollowRequest{
		ID:              followID,
		URI:             followURI,
		AccountID:       requestingAccount.ID,
		Account:         requestingAccount,
		TargetAccountID: form.ID,
		TargetAccount:   targetAccount,
		ShowReblogs:     form.Reblogs,
		Notify:          form.Notify,
	}

	if err := p.state.DB.PutFollowRequest(ctx, fr); err != nil {
		err = fmt.Errorf("FollowCreate: error creating follow request in db: %s", err)
		return nil, gtserror.NewErrorInternalError(err)
	}

	if targetAccount.IsLocal() && !*targetAccount.Locked {
		// If the target account is local and not locked,
		// we can already accept the follow request and
		// skip any further processing.
		//
		// Because we know the requestingAccount is also
		// local, we don't need to federate the accept out.
		if _, err := p.state.DB.AcceptFollowRequest(ctx, requestingAccount.ID, form.ID); err != nil {
			err = fmt.Errorf("FollowCreate: error accepting follow request for local unlocked account: %w", err)
			return nil, gtserror.NewErrorInternalError(err)
		}
	} else if targetAccount.IsRemote() {
		// Otherwise we leave the follow request as it is,
		// and we handle the rest of the process async.
		p.state.Workers.EnqueueClientAPI(ctx, messages.FromClientAPI{
			APObjectType:   ap.ActivityFollow,
			APActivityType: ap.ActivityCreate,
			GTSModel:       fr,
			OriginAccount:  requestingAccount,
			TargetAccount:  targetAccount,
		})
	}

	return p.RelationshipGet(ctx, requestingAccount, form.ID)
}

// FollowRemove handles the removal of a follow/follow request to an account, either remote or local.
func (p *Processor) FollowRemove(ctx context.Context, requestingAccount *gtsmodel.Account, targetAccountID string) (*apimodel.Relationship, gtserror.WithCode) {
	targetAccount, errWithCode := p.getFollowTarget(ctx, requestingAccount.ID, targetAccountID)
	if errWithCode != nil {
		return nil, errWithCode
	}

	// Unfollow and deal with side effects.
	msgs, err := p.unfollow(ctx, requestingAccount, targetAccount)
	if err != nil {
		return nil, gtserror.NewErrorNotFound(fmt.Errorf("FollowRemove: account %s not found in the db: %s", targetAccountID, err))
	}

	// Batch queue accreted client api messages.
	p.state.Workers.EnqueueClientAPI(ctx, msgs...)

	return p.RelationshipGet(ctx, requestingAccount, targetAccountID)
}

/*
	Utility functions.
*/

// updateFollow is a utility function for updating an existing
// follow or followRequest with the parameters provided in the
// given form. If nothing changes, this function is a no-op and
// will just return the existing relationship between follow
// origin and follow target account.
func (p *Processor) updateFollow(
	ctx context.Context,
	requestingAccount *gtsmodel.Account,
	form *apimodel.AccountFollowRequest,
	currentShowReblogs *bool,
	currentNotify *bool,
	update func(...string) error,
) (*apimodel.Relationship, gtserror.WithCode) {

	if form.Reblogs == nil && form.Notify == nil {
		// There's nothing to update.
		return p.RelationshipGet(ctx, requestingAccount, form.ID)
	}

	// Including "updated_at", max 3 columns may change.
	columns := make([]string, 0, 3)

	// Check what we need to update (if anything).
	if newReblogs := form.Reblogs; newReblogs != nil && *newReblogs != *currentShowReblogs {
		*currentShowReblogs = *newReblogs
		columns = append(columns, "show_reblogs")
	}

	if newNotify := form.Notify; newNotify != nil && *newNotify != *currentNotify {
		*currentNotify = *newNotify
		columns = append(columns, "notify")
	}

	if len(columns) == 0 {
		// Nothing actually changed.
		return p.RelationshipGet(ctx, requestingAccount, form.ID)
	}

	if err := update(columns...); err != nil {
		err = fmt.Errorf("updateFollow: error updating existing follow (request): %w", err)
		return nil, gtserror.NewErrorInternalError(err)
	}

	return p.RelationshipGet(ctx, requestingAccount, form.ID)
}

// getFollowTarget is a convenience function which:
//   - Checks if account is trying to follow/unfollow itself.
//   - Returns not found if there's a block in place between accounts.
//   - Returns target account according to its id.
func (p *Processor) getFollowTarget(ctx context.Context, requestingAccountID string, targetAccountID string) (*gtsmodel.Account, gtserror.WithCode) {
	// Account can't follow or unfollow itself.
	if requestingAccountID == targetAccountID {
		err := errors.New("account can't follow or unfollow itself")
		return nil, gtserror.NewErrorNotAcceptable(err)
	}

	// Do nothing if a block exists in either direction between accounts.
	if blocked, err := p.state.DB.IsEitherBlocked(ctx, requestingAccountID, targetAccountID); err != nil {
		err = fmt.Errorf("db error checking block between accounts: %w", err)
		return nil, gtserror.NewErrorInternalError(err)
	} else if blocked {
		err = errors.New("block exists between accounts")
		return nil, gtserror.NewErrorNotFound(err)
	}

	// Ensure target account retrievable.
	targetAccount, err := p.state.DB.GetAccountByID(ctx, targetAccountID)
	if err != nil {
		if !errors.Is(err, db.ErrNoEntries) {
			// Real db error.
			err = fmt.Errorf("db error looking for target account %s: %w", targetAccountID, err)
			return nil, gtserror.NewErrorInternalError(err)
		}
		// Account not found.
		err = fmt.Errorf("target account %s not found in the db", targetAccountID)
		return nil, gtserror.NewErrorNotFound(err, err.Error())
	}

	return targetAccount, nil
}

// unfollow is a convenience function for having requesting account
// unfollow (and un follow request) target account, if follows and/or
// follow requests exist.
//
// If a follow and/or follow request was removed this way, one or two
// messages will be returned which should then be processed by a client
// api worker.
func (p *Processor) unfollow(ctx context.Context, requestingAccount *gtsmodel.Account, targetAccount *gtsmodel.Account) ([]messages.FromClientAPI, error) {
	var msgs []messages.FromClientAPI

	// Get follow from requesting account to target account.
	follow, err := p.state.DB.GetFollow(ctx, requestingAccount.ID, targetAccount.ID)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		err = fmt.Errorf("unfollow: error getting follow from %s targeting %s: %w", requestingAccount.ID, targetAccount.ID, err)
		return nil, err
	}

	if follow != nil {
		// Delete known follow from database with ID.
		err = p.state.DB.DeleteFollowByID(ctx, follow.ID)
		if err != nil {
			if !errors.Is(err, db.ErrNoEntries) {
				err = fmt.Errorf("unfollow: error deleting request from %s targeting %s: %w", requestingAccount.ID, targetAccount.ID, err)
				return nil, err
			}

			// If err == db.ErrNoEntries here then it
			// indicates a race condition with another
			// unfollow for the same requester->target.
			return msgs, nil
		}

		// Follow status changed, process side effects.
		msgs = append(msgs, messages.FromClientAPI{
			APObjectType:   ap.ActivityFollow,
			APActivityType: ap.ActivityUndo,
			GTSModel: &gtsmodel.Follow{
				AccountID:       requestingAccount.ID,
				TargetAccountID: targetAccount.ID,
				URI:             follow.URI,
			},
			OriginAccount: requestingAccount,
			TargetAccount: targetAccount,
		})
	}

	// Get follow request from requesting account to target account.
	followReq, err := p.state.DB.GetFollowRequest(ctx, requestingAccount.ID, targetAccount.ID)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		err = fmt.Errorf("unfollow: error getting follow request from %s targeting %s: %w", requestingAccount.ID, targetAccount.ID, err)
		return nil, err
	}

	if followReq != nil {
		// Delete known follow request from database with ID.
		err = p.state.DB.DeleteFollowRequestByID(ctx, followReq.ID)
		if err != nil {
			if !errors.Is(err, db.ErrNoEntries) {
				err = fmt.Errorf("unfollow: error deleting follow request from %s targeting %s: %w", requestingAccount.ID, targetAccount.ID, err)
				return nil, err
			}

			// If err == db.ErrNoEntries here then it
			// indicates a race condition with another
			// unfollow for the same requester->target.
			return msgs, nil
		}

		// Follow status changed, process side effects.
		msgs = append(msgs, messages.FromClientAPI{
			APObjectType:   ap.ActivityFollow,
			APActivityType: ap.ActivityUndo,
			GTSModel: &gtsmodel.Follow{
				AccountID:       requestingAccount.ID,
				TargetAccountID: targetAccount.ID,
				URI:             followReq.URI,
			},
			OriginAccount: requestingAccount,
			TargetAccount: targetAccount,
		})
	}

	return msgs, nil
}
