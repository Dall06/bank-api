# Estrategia de Trabajo Colaborativo (IA & Humano)

Este documento detalla la dinámica, estrategias y metodologías aplicadas durante la construcción del proyecto `bank-api`. El desarrollo se llevó a cabo bajo un enfoque de **Pair Programming de Próxima Generación**, combinando la intuición y el contexto de negocio de un Ingeniero Humano con la capacidad de análisis y generación de código de Agentes de IA.

## 1. La Dinámica de Trabajo (Roles)

* **Ingeniero Humano (Tech Lead / Operador del Entorno):** 
  * Dueño del clúster local y del orquestador (`docker-compose`).
  * Proveedor del contexto y prioridades de negocio (ej. "Desactiva JWT para no depender de la DB").
  * Revisor final de los cambios propuestos (Pull Request Reviewer en tiempo real).
  * Ejecutor de los despliegues locales y scripts pesados de infraestructura.

* **Agente de IA (Desarrollador / Arquitecto / Debugger):**
  * **Andamiaje Rápido:** Generador de código repetitivo (Boilerplate) siguiendo Arquitectura Hexagonal.
  * **Análisis Forense (Debugging):** Análisis en profundidad de trazas y logs de errores 500 (ej. descubriendo bloqueos de Kafka o el bug de `redis: nil`).
  * **Refactorización Segura:** Edición in-place de archivos específicos cuidando de no romper componentes acoplados, guiado por TDD.

## 2. Estrategias Metodológicas Adoptadas

### 2.1. "Medir dos veces, cortar una" (Análisis Previo)
Antes de ejecutar cambios destructivos o escribir código masivo, la IA estaba instruida para leer el código base (`grep`, lectura de main, lectura de middlewares), identificar las dependencias y rastrear exactamente cómo un cambio en el Gateway impactaría en el microservicio de Transacciones. 

### 2.2. Debugging Basado en Evidencia (Logs First)
En lugar de lanzar teorías al aire cuando los endpoints arrojaban un *500 Internal Error* o un *404 Not Found*, la estrategia siempre fue **ir a los logs de Docker del contenedor fallido**. 
Por ejemplo:
1. El Gateway devolvía 500.
2. Fuimos a los logs de Transactions y descubrimos un 404 interno.
3. Se revisó el código del Gateway y se encontró el bug del prefijo `/api/v1`.
Este ciclo de *Reproducir -> Observar Logs -> Reparar* redujo drásticamente el tiempo de resolución.

### 2.3. Respeto al Ownership del Ecosistema Local
La IA no reiniciaba los servicios a sus espaldas ni reconstruía contenedores sin avisar. Se estableció un protocolo donde la IA informaba que el código había sido reparado, pero le pasaba la batuta al humano para ejecutar `task up` y el subsecuente `curl`. Esto previno sobrecarga computacional y mantuvo al humano en absoluto control.

### 2.4. Refinamiento Evolutivo del Código (Refactoring Constante)
El proyecto no nació perfecto. Creció bajo iteraciones:
* **Fase 1:** Sincronía pura, que provocó cuellos de botella y timeouts esperando al proveedor.
* **Fase 2:** Inyección de Caché Redis.
* **Fase 3:** Se observó que publicar a Kafka bloqueaba el HTTP final. 
* **Fase 4:** Migración a patrones *Fire-and-Forget* con Goroutines asíncronas para retornar un 201 instantáneo al cliente.

## 3. Conclusión de la Sinergia
El resultado es un sistema que habría tomado semanas construir de cero por una sola persona. Gracias a la división de responsabilidades (Humano dirige la orquesta y plantea los retos lógicos; IA investiga, teclea, optimiza y documenta a velocidad luz), se logró entregar un sistema robusto, con pruebas unitarias, cacheado y asíncrono en una fracción del tiempo.
