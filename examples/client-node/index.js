const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const fs = require('fs');
const path = require('path');

// 1. Cargar el archivo .proto dinámicamente
const PROTO_PATH = path.resolve(__dirname, '../../api/v1/clinical_record.proto');
const INCLUDE_PATH = path.resolve(__dirname, '../../third_party');

const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
  includeDirs: [INCLUDE_PATH, path.resolve(__dirname, '../../')]
});

const clinicalProto = grpc.loadPackageDefinition(packageDefinition).vinca.clinical.v1;

function main() {
  console.log('Iniciando cliente de prueba VINCULA Salud (Node.js)...');

  // 2. Leer certificados mTLS
  const certsDir = path.resolve(__dirname, '../../certs');
  
  let caCert, clientKey, clientCert;
  try {
    caCert = fs.readFileSync(path.join(certsDir, 'ca.crt'));
    clientKey = fs.readFileSync(path.join(certsDir, 'client.key'));
    clientCert = fs.readFileSync(path.join(certsDir, 'client.crt'));
  } catch (error) {
    console.error('❌ Error leyendo certificados. Asegúrate de ejecutar este script desde examples/client-node/ y de que existan en la carpeta certs/.');
    process.exit(1);
  }

  // 3. Crear credenciales mTLS
  const credentials = grpc.credentials.createSsl(caCert, clientKey, clientCert);

  // 4. Conectar al servidor
  const client = new clinicalProto.ClinicalRecordService(
    'localhost:50051',
    credentials
  );

  const patientRun = '12345678-9';

  console.log(`\nConsultando resumen del paciente ${patientRun}...`);
  
  // 5. Invocar el método gRPC
  client.GetPatientSummary({ patient_run: patientRun, requester_hospital_id: 'HOSP-NODE' }, (err, response) => {
    if (err) {
      console.error(`❌ Fallo al obtener resumen: ${err.message}`);
      return;
    }

    console.log(`✅ Resumen obtenido para RUN: ${response.patient_run}`);
    console.log(`   Alergias activas:`, response.active_allergies || []);
    console.log(`   Diagnósticos activos:`, response.active_diagnoses || []);
    console.log(`   Medicamentos activos:`, response.active_medications || []);
    
    if (response.last_update) {
        console.log(`   Última actualización:`, new Date(response.last_update.seconds * 1000).toISOString());
    }
  });
}

main();
