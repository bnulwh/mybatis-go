package orm

import (
	"context"
	"testing"
)

func Test_ReadWriteSplitting_Config(t *testing.T) {
	SetReadWriteSplitting(true)
	if !ReadWriteSplittingEnabled() {
		t.Error("expected ReadWriteSplittingEnabled = true")
	}
	SetReadWriteSplitting(false)
	if ReadWriteSplittingEnabled() {
		t.Error("expected ReadWriteSplittingEnabled = false")
	}
}

func Test_ReplicaNames(t *testing.T) {
	SetReplicaNames([]string{"replica1", "replica2"})
	names := GetReplicaNames()
	if len(names) != 2 || names[0] != "replica1" || names[1] != "replica2" {
		t.Errorf("replica names = %v, want [replica1 replica2]", names)
	}
	SetReplicaNames(nil)
	names = GetReplicaNames()
	if len(names) != 0 {
		t.Errorf("expected empty, got %v", names)
	}
}

func Test_RouteReplica_Disabled(t *testing.T) {
	SetReadWriteSplitting(false)
	db := routeReplica()
	if db != nil {
		t.Error("expected nil when disabled")
	}
}

func Test_RouteReplica_NoReplicas(t *testing.T) {
	SetReadWriteSplitting(true)
	defer SetReadWriteSplitting(false)
	SetReplicaNames(nil)
	db := routeReplica()
	if db != nil {
		t.Error("expected nil with no replicas")
	}
}

func Test_RouteReadDB_Disabled(t *testing.T) {
	SetReadWriteSplitting(false)
	ctx := routeReadDB(context.Background())
	if ctx.Value(rwRouteKey{}) != nil {
		t.Error("expected no route key when disabled")
	}
}

func Test_RouteReadDB_InTransaction(t *testing.T) {
	SetReadWriteSplitting(true)
	defer SetReadWriteSplitting(false)
	SetReplicaNames([]string{"replica1"})
	defer SetReplicaNames(nil)

	if gDbConn != nil {
		gDbConn.setCurTx(&Transaction{})
		ctx := routeReadDB(context.Background())
		if ctx.Value(rwRouteKey{}) != nil {
			t.Error("expected no routing in transaction")
		}
		gDbConn.clearCurTx(nil)
	}
}

func Test_RouteReadDB_WithTxInContext(t *testing.T) {
	SetReadWriteSplitting(true)
	defer SetReadWriteSplitting(false)
	SetReplicaNames([]string{"replica1"})
	defer SetReplicaNames(nil)

	txCtx := context.WithValue(context.Background(), txContextKey{}, &Transaction{})
	ctx := routeReadDB(txCtx)
	if ctx.Value(rwRouteKey{}) != nil {
		t.Error("expected no routing when tx in context")
	}
}

func Test_RoutedDB_Default(t *testing.T) {
	ctx := context.Background()
	db := routedDB(ctx)
	if db != gDbConn {
		t.Error("expected gDbConn as default")
	}
}

func Test_RoutedDB_WithRoute(t *testing.T) {
	mockDB := &DB{}
	ctx := contextWithRoute(context.Background(), mockDB)
	db := routedDB(ctx)
	if db != mockDB {
		t.Error("expected routed DB")
	}
}

func Test_PickReplica_NoReplicas(t *testing.T) {
	SetReplicaNames(nil)
	_, err := PickReplica()
	if err == nil {
		t.Error("expected error with no replicas")
	}
}

func Test_InitReplicaDatasources_Empty(t *testing.T) {
	cm := map[string]string{}
	initReplicaDatasources(cm)
	names := GetReplicaNames()
	if len(names) != 0 {
		t.Errorf("expected empty, got %v", names)
	}
}
