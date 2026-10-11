package com.example.shop.payments;

public record ChargeResult(String chargeId, Status status, String declineCode) {

    public enum Status { SUCCEEDED, DECLINED, PENDING }

    public static ChargeResult succeeded(String chargeId) {
        return new ChargeResult(chargeId, Status.SUCCEEDED, null);
    }

    public static ChargeResult declined(String declineCode) {
        return new ChargeResult(null, Status.DECLINED, declineCode);
    }

    public boolean isSuccessful() {
        return status == Status.SUCCEEDED;
    }
}
