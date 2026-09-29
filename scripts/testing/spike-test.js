import http from "k6/http";
import { check, sleep } from "k6";
import { uuidv4 } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";

export const options = {
  scenarios: {
    tip_load_test: {
      executor: "constant-arrival-rate",
      rate: 2000, // Target: 2,000 requests per second!
      timeUnit: "1s",
      duration: "30s", // Run for 30 seconds
      preAllocatedVUs: 200, // Start with 200 VUs ready
      maxVUs: 2500, // Allow k6 to scale up VUs if needed to maintain 2k rate
    },
  },
};

const BASE_URL = "http://localhost:8085"; // Or http://host.docker.internal:8085 if running via Docker without host network

export default function () {
  const payload = JSON.stringify({
    sender_id: "sender_" + Math.floor(Math.random() * 100),
    creator_id: "creator_" + Math.floor(Math.random() * 50),
    idempotency_key: uuidv4(), // Ensures unique key per request to pass validation
    amount: Math.floor(Math.random() * 94) + 7,
  });

  const params = { headers: { "Content-Type": "application/json" } };
  let res = http.post(`${BASE_URL}/tip-transfer`, payload, params);

  check(res, { "status is 200": (r) => r.status === 200 });
  sleep(0.1);
}
