import grpc from 'k6/net/grpc';
import { check, sleep } from 'k6';

const client = new grpc.Client();
client.load(['../../api/v1'], 'clinical_record.proto');

export const options = {
  stages: [
    { duration: '10s', target: 5 },  // Ramp-up a 5 VUs (Usuarios Virtuales)
    { duration: '20s', target: 10 }, // Escalar a 10 VUs
    { duration: '10s', target: 0 },  // Ramp-down a 0
  ],
  // Configuración mTLS (k6 cargará estos certificados si se proveen opciones tlsAuth en el runner)
  insecureSkipTlsVerify: true,
};

// Como k6 lee los certificados TLS desde flags de línea de comandos o globalmente,
// el runner debe ejecutarse con:
// k6 run --ssl-client-cert certs/client.crt --ssl-client-key certs/client.key tests/load/grpc_load_test.js

export default () => {
  client.connect('localhost:50051', {
    plaintext: false
  });

  const data = { patient_run: 'LOAD-TEST-RUN', requester_hospital_id: 'HOSP-LOAD' };
  const response = client.invoke('vinca.clinical.v1.ClinicalRecordService/GetPatientSummary', data);

  check(response, {
    'status is OK': (r) => r && r.status === grpc.StatusOK,
  });

  client.close();
  sleep(1);
};
