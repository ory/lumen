package com.example.shop.payments;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;

public class StripePaymentGateway implements PaymentGateway {

    private static final URI CHARGES_URI = URI.create("https://api.stripe.com/v1/charges");
    private static final URI REFUNDS_URI = URI.create("https://api.stripe.com/v1/refunds");

    private final HttpClient http;
    private final ObjectMapper mapper;
    private final String secretKey;

    public StripePaymentGateway(HttpClient http, ObjectMapper mapper, String secretKey) {
        this.http = http;
        this.mapper = mapper;
        this.secretKey = secretKey;
    }

    @Override
    public ChargeResult charge(String customerId, long amountCents, String currency, String idempotencyKey) {
        String form = "customer=" + customerId + "&amount=" + amountCents + "&currency=" + currency;
        HttpRequest request = HttpRequest.newBuilder(CHARGES_URI)
            .timeout(Duration.ofSeconds(30))
            .header("Authorization", "Bearer " + secretKey)
            .header("Idempotency-Key", idempotencyKey)
            .header("Content-Type", "application/x-www-form-urlencoded")
            .POST(HttpRequest.BodyPublishers.ofString(form, StandardCharsets.UTF_8))
            .build();
        try {
            HttpResponse<String> response = http.send(request, HttpResponse.BodyHandlers.ofString());
            JsonNode body = mapper.readTree(response.body());
            if (response.statusCode() == 402) {
                return ChargeResult.declined(body.path("error").path("decline_code").asText());
            }
            if (response.statusCode() >= 300) {
                throw new PaymentException("stripe returned HTTP " + response.statusCode());
            }
            return ChargeResult.succeeded(body.path("id").asText());
        } catch (IOException e) {
            throw new PaymentException("stripe request failed", e);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new PaymentException("stripe request interrupted", e);
        }
    }

    @Override
    public void refund(String chargeId, long amountCents) {
        String form = "charge=" + chargeId + "&amount=" + amountCents;
        HttpRequest request = HttpRequest.newBuilder(REFUNDS_URI)
            .header("Authorization", "Bearer " + secretKey)
            .POST(HttpRequest.BodyPublishers.ofString(form, StandardCharsets.UTF_8))
            .build();
        try {
            HttpResponse<Void> response = http.send(request, HttpResponse.BodyHandlers.discarding());
            if (response.statusCode() >= 300) {
                throw new PaymentException("stripe refund returned HTTP " + response.statusCode());
            }
        } catch (IOException e) {
            throw new PaymentException("stripe refund failed", e);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new PaymentException("stripe refund interrupted", e);
        }
    }
}
