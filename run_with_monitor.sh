#!/bin/bash

# Timeout absoluto: 180s
TIMEOUT=180
ELAPSED=0

# Inicia o processo em background e guarda PID
"$@" &
PID=$!

# Last modtime file placeholder
LAST_MOD_FILE=""

while [ $ELAPSED -lt $TIMEOUT ]; do
  # Verifica se processo ainda existe
  if ! kill -0 $PID 2>/dev/null; then
    exit 0
  fi

  # Verifica atividade de arquivo com lsof + find
  ACTIVE=0
  if lsof -p $PID >/dev/null 2>&1; then
    MODIFIED=$(find . -mmin -0.1 2>/dev/null | head -n1)
    if [ -n "$MODIFIED" ]; then
      LAST_MOD_FILE="$MODIFIED"
      ACTIVE=1
    fi
  fi

  # Se lsof falhar (ex: processo terminou), sai com sucesso
  if [ $ACTIVE -eq 0 ] && [ -z "$LAST_MOD_FILE" ]; then
    # Apenas se já tivermos registrado algo anteriormente, considera estagnação
    # Mas para segurança, não mata imediatamente — espera 2 ciclos sem atividade
    # Simplificação: considera estagnação após 2 verificações seguidas sem modificações
    STAGNATION_COUNT=$((STAGNATION_COUNT + 1))
    if [ $STAGNATION_COUNT -ge 2 ]; then
      kill -SIGTERM $PID 2>/dev/null
      sleep 2
      if kill -0 $PID 2>/dev/null; then
        kill -SIGKILL $PID 2>/dev/null
      fi
      exit 1
    fi
  else
    STAGNATION_COUNT=0
  fi

  sleep 1
  ELAPSED=$((ELAPSED + 1))
done

# Timeout absoluto atingido: mata
kill -SIGTERM $PID 2>/dev/null
sleep 2
if kill -0 $PID 2>/dev/null; then
  kill -SIGKILL $PID 2>/dev/null
fi
exit 1
