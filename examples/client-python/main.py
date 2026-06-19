import grpc
import os
import time
from datetime import datetime

# Asume que los archivos generados están en el mismo directorio
import clinical_record_pb2
import clinical_record_pb2_grpc

def run():
    print("Iniciando cliente de prueba VINCULA Salud (Python)...")

    # Rutas relativas a la raíz del proyecto
    cert_dir = "../../certs"
    ca_cert_path = os.path.join(cert_dir, "ca.crt")
    client_cert_path = os.path.join(cert_dir, "client.crt")
    client_key_path = os.path.join(cert_dir, "client.key")

    try:
        with open(ca_cert_path, "rb") as f:
            ca_cert = f.read()
        with open(client_key_path, "rb") as f:
            client_key = f.read()
        with open(client_cert_path, "rb") as f:
            client_cert = f.read()
    except FileNotFoundError as e:
        print(f"Error leyendo certificados: {e}")
        print("Asegúrate de ejecutar este script desde 'examples/client-python/' y haber generado los certificados en 'certs/'")
        return

    # 1. Crear credenciales mTLS
    credentials = grpc.ssl_channel_credentials(
        root_certificates=ca_cert,
        private_key=client_key,
        certificate_chain=client_cert
    )

    # 2. Conectar al servidor
    channel = grpc.secure_channel('localhost:50051', credentials)
    client = clinical_record_pb2_grpc.ClinicalRecordServiceStub(channel)

    patient_run = "12345678-9"

    # 3. Consultar el resumen del paciente
    print(f"\nConsultando resumen del paciente {patient_run}...")
    try:
        req = clinical_record_pb2.GetPatientSummaryRequest(
            patient_run=patient_run,
            requester_hospital_id="HOSP-PYTHON"
        )
        response = client.GetPatientSummary(req, timeout=5)
        
        print(f"✅ Resumen obtenido para RUN: {response.patient_run}")
        print(f"   Alergias activas: {list(response.active_allergies)}")
        print(f"   Diagnósticos activos: {list(response.active_diagnoses)}")
        print(f"   Medicamentos activos: {list(response.active_medications)}")
        
        if response.HasField('last_update'):
            dt = response.last_update.ToDatetime()
            print(f"   Última actualización: {dt.isoformat()}")
            
    except grpc.RpcError as e:
        print(f"❌ Fallo al obtener resumen: {e.details()} (Código: {e.code()})")

if __name__ == '__main__':
    run()
