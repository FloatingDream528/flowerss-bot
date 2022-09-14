package core

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/indes/flowerss-bot/internal/model"
	"github.com/indes/flowerss-bot/internal/storage"
	"github.com/indes/flowerss-bot/internal/storage/mock"
)

type mockStorage struct {
	User         *mock.MockUser
	Content      *mock.MockContent
	Source       *mock.MockSource
	Subscription *mock.MockSubscription
	Ctrl         *gomock.Controller
}

func getTestCore(t *testing.T) (*Core, *mockStorage) {
	ctrl := gomock.NewController(t)

	s := &mockStorage{
		Subscription: mock.NewMockSubscription(ctrl),
		User:         mock.NewMockUser(ctrl),
		Content:      mock.NewMockContent(ctrl),
		Source:       mock.NewMockSource(ctrl),
		Ctrl:         ctrl,
	}
	c := NewCore(s.User, s.Content, s.Source, s.Subscription)
	return c, s
}

func TestCore_AddSubscription(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()

	userID := int64(1)
	sourceID := uint(101)
	t.Run(
		"exist error", func(t *testing.T) {
			s.Subscription.EXPECT().SubscriptionExist(ctx, userID, sourceID).Return(false, errors.New("err")).Times(1)
			err := c.AddSubscription(ctx, userID, sourceID)
			assert.Error(t, err)
		},
	)

	t.Run(
		"exist subscription", func(t *testing.T) {
			s.Subscription.EXPECT().SubscriptionExist(ctx, userID, sourceID).Return(true, nil).Times(1)
			err := c.AddSubscription(ctx, userID, sourceID)
			assert.Equal(t, ErrSubscriptionExist, err)
		},
	)

	t.Run(
		"subscribe fail", func(t *testing.T) {
			s.Subscription.EXPECT().SubscriptionExist(ctx, userID, sourceID).Return(false, nil).Times(1)
			s.Subscription.EXPECT().AddSubscription(ctx, gomock.Any()).Return(errors.New("err")).Times(1)

			err := c.AddSubscription(ctx, userID, sourceID)
			assert.Error(t, err)
		},
	)

	t.Run(
		"subscribe ok", func(t *testing.T) {
			s.Subscription.EXPECT().SubscriptionExist(ctx, userID, sourceID).Return(false, nil).Times(1)
			s.Subscription.EXPECT().AddSubscription(ctx, gomock.Any()).Return(nil).Times(1)

			err := c.AddSubscription(ctx, userID, sourceID)
			assert.Nil(t, err)
		},
	)
}

func TestCore_GetUserSubscribedSources(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()

	userID := int64(1)
	sourceID1 := uint(101)
	sourceID2 := uint(102)
	subscriptionsResult := &storage.GetSubscriptionsResult{
		Subscriptions: []*model.Subscribe{
			&model.Subscribe{SourceID: sourceID1},
			&model.Subscribe{SourceID: sourceID2},
		},
	}
	t.Run(
		"subscription err", func(t *testing.T) {
			s.Subscription.EXPECT().GetSubscriptionsByUserID(ctx, userID, gomock.Any()).Return(
				nil, errors.New("err"),
			)

			sources, err := c.GetUserSubscribedSources(ctx, userID)
			assert.Error(t, err)
			assert.Nil(t, sources)
		},
	)

	t.Run(
		"source err", func(t *testing.T) {
			s.Subscription.EXPECT().GetSubscriptionsByUserID(ctx, userID, gomock.Any()).Return(
				subscriptionsResult, nil,
			)

			s.Source.EXPECT().GetSource(ctx, sourceID1).Return(
				nil, errors.New("err"),
			).Times(1)
			s.Source.EXPECT().GetSource(ctx, gomock.Any()).Return(
				&model.Source{}, nil,
			)

			sources, err := c.GetUserSubscribedSources(ctx, userID)
			assert.Nil(t, err)
			assert.Equal(t, len(subscriptionsResult.Subscriptions)-1, len(sources))
		},
	)

	t.Run(
		"source success", func(t *testing.T) {
			s.Subscription.EXPECT().GetSubscriptionsByUserID(ctx, userID, gomock.Any()).Return(
				subscriptionsResult, nil,
			)

			s.Source.EXPECT().GetSource(ctx, gomock.Any()).Return(
				&model.Source{}, nil,
			).Times(len(subscriptionsResult.Subscriptions))

			sources, err := c.GetUserSubscribedSources(ctx, userID)
			assert.Nil(t, err)
			assert.Equal(t, len(subscriptionsResult.Subscriptions), len(sources))
		},
	)
}

func TestCore_Unsubscribe(t *testing.T) {
	c, s := getTestCore(t)
	defer s.Ctrl.Finish()
	ctx := context.Background()

	userID := int64(1)
	sourceID1 := uint(101)

	t.Run(
		"SubscriptionExist err", func(t *testing.T) {
			s.Subscription.EXPECT().SubscriptionExist(ctx, userID, sourceID1).Return(
				false, errors.New("err"),
			).Times(1)
			err := c.Unsubscribe(ctx, userID, sourceID1)
			assert.Error(t, err)
		},
	)

	t.Run(
		"subscription not exist", func(t *testing.T) {
			s.Subscription.EXPECT().SubscriptionExist(ctx, userID, sourceID1).Return(
				false, nil,
			).Times(1)
			err := c.Unsubscribe(ctx, userID, sourceID1)
			assert.Equal(t, ErrSubscriptionNotExist, err)
		},
	)

	s.Subscription.EXPECT().SubscriptionExist(ctx, gomock.Any(), gomock.Any()).Return(
		true, nil,
	).AnyTimes()

	t.Run(
		"unsubscribe failed", func(t *testing.T) {
			s.Subscription.EXPECT().DeleteSubscription(ctx, userID, sourceID1).Return(
				int64(1), errors.New("err"),
			).Times(1)
			err := c.Unsubscribe(ctx, userID, sourceID1)
			assert.Error(t, err)
		},
	)

	s.Subscription.EXPECT().DeleteSubscription(ctx, gomock.Any(), gomock.Any()).Return(
		int64(1), nil,
	).AnyTimes()

	t.Run(
		"count subs", func(t *testing.T) {
			s.Subscription.EXPECT().CountSourceSubscriptions(ctx, sourceID1).Return(
				int64(1), errors.New("err"),
			).Times(1)
			err := c.Unsubscribe(ctx, userID, sourceID1)
			assert.Error(t, err)

			s.Subscription.EXPECT().CountSourceSubscriptions(ctx, sourceID1).Return(
				int64(1), nil,
			).Times(1)
			err = c.Unsubscribe(ctx, userID, sourceID1)
			assert.Nil(t, err)
		},
	)

	s.Subscription.EXPECT().CountSourceSubscriptions(ctx, gomock.Any()).Return(
		int64(0), nil,
	).AnyTimes()

	t.Run(
		"remove source", func(t *testing.T) {
			s.Source.EXPECT().Delete(ctx, sourceID1).Return(
				errors.New("err"),
			).Times(1)

			err := c.Unsubscribe(ctx, userID, sourceID1)
			assert.Error(t, err)

			s.Source.EXPECT().Delete(ctx, sourceID1).Return(nil).AnyTimes()

			s.Content.EXPECT().DeleteSourceContents(ctx, sourceID1).Return(int64(0), errors.New("err")).Times(1)
			err = c.Unsubscribe(ctx, userID, sourceID1)
			assert.Error(t, err)

			s.Content.EXPECT().DeleteSourceContents(ctx, sourceID1).Return(int64(1), nil).Times(1)
			err = c.Unsubscribe(ctx, userID, sourceID1)
			assert.Nil(t, err)
		},
	)
}
