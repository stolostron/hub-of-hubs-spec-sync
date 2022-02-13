package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	subv1 "github.com/open-cluster-management/multicloud-operators-subscription/pkg/apis/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const NamespaceToSkip = "open-cluster-management"

func addSubscriptionController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&subv1.Subscription{}).
		Complete(&genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("subscriptions-spec-syncer"),
			tableName:              "subscriptions",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/subscription-cleanup",
			createInstance:         func() client.Object { return &subv1.Subscription{} },
			cleanStatus:            cleanSubscriptionStatus,
			areEqual:               areSubscriptionsEqual,
			shouldProcess:          namespaceFilterFunc,
		})
	if err != nil {
		return fmt.Errorf("failed to add Subscription Controller to the manager: %w", err)
	}

	return nil
}

func areSubscriptionsEqual(instance1, instance2 client.Object) bool {
	// TODO: subscription come out as not equal because of package override field, check if it matters.
	// TODO: subscription keeps entering reconcile because placement keeps changing.
	annotationMatch := equality.Semantic.DeepEqual(instance1.GetAnnotations(), instance2.GetAnnotations())

	subscription1, ok1 := instance1.(*subv1.Subscription)
	subscription2, ok2 := instance2.(*subv1.Subscription)
	specMatch := ok1 && ok2 && equality.Semantic.DeepEqual(subscription1.Spec, subscription2.Spec)

	return annotationMatch && specMatch
}

func cleanSubscriptionStatus(instance client.Object) {
	subscription, ok := instance.(*subv1.Subscription)
	if !ok {
		panic("wrong instance passed to cleanConfigStatus: not subv1.Subscription")
	}

	subscription.Status = subv1.SubscriptionStatus{}
}

func namespaceFilterFunc(instance client.Object) bool {
	return !(instance.GetNamespace() == NamespaceToSkip)
}
