import { useState, useEffect } from "react";
import { DayPicker } from "@daypicker/react";
import "@daypicker/react/style.css";
import axios from "axios";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  ReferenceLine,
} from "recharts";
import { DESTINATION_LABELS } from "../constants";

const API = import.meta.env.VITE_API_URL;

const userTZ = Intl.DateTimeFormat().resolvedOptions().timeZone;

const formatDate = (dateStr, options) => {
  if (!dateStr) return "";

  const date = /^\d{4}-\d{2}-\d{2}$/.test(dateStr)
    ? new Date(`${dateStr}T12:00:00`)
    : new Date(dateStr);

  return new Intl.DateTimeFormat(undefined, {
    timeZone: userTZ,
    ...options,
  }).format(date);
};

const cleanPriceData = (prices) =>
  prices
    ?.filter((p) => p.date && Number(p.price) > 0)
    .map((p) => ({
      date: p.date,
      price: Number(p.price),
    })) || [];

const parseDateOnly = (dateStr) => {
  const [year, month, day] = dateStr.split("-").map(Number);

  return new Date(year, month - 1, day);
};

const toDateKey = (date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");

  return `${year}-${month}-${day}`;
};

export default function PriceChart({ route }) {
  const routeKey = route
    ? `${route.origin}-${route.destination}`
    : "";

  const [departureResult, setDepartureResult] = useState({
    routeKey: "",
    data: [],
    error: "",
  });

  const [historyResult, setHistoryResult] = useState({
    key: "",
    data: [],
    error: "",
  });

  const [view, setView] = useState({
    routeKey: "",
    mode: "depart",
    departDate: null,
  });

  /*
   * ------------------------------------------------------------
   * TODAY / CURRENT MONTH
   * ------------------------------------------------------------
   */
  const today = new Date();
  today.setHours(0, 0, 0, 0);

  const currentMonth = new Date(
    today.getFullYear(),
    today.getMonth(),
    1,
  );

  /*
   * If the route changes, treat it as a fresh departure-price view.
   */
  const currentView =
    view.routeKey === routeKey
      ? view
      : {
          routeKey,
          mode: "depart",
          departDate: null,
        };

  const mode = currentView.mode;
  const selectedDepartDate = currentView.departDate;

  /*
   * Departure prices belonging to the current route.
   */
  const departures =
    departureResult.routeKey === routeKey
      ? departureResult.data
      : [];

  /*
   * All dates for which we have fare data.
   */
  const availableDateKeys = new Set(
    departures.map((departure) => departure.date),
  );

  /*
   * Only keep departure dates from today onward.
   */
  const futureDepartures = departures.filter(
    (departure) =>
      parseDateOnly(departure.date) >= today,
  );

  /*
   * Date selected in the DayPicker calendar.
   */
  const selectedCalendarDate = selectedDepartDate
    ? parseDateOnly(selectedDepartDate)
    : undefined;

  /*
   * Last future departure date determines how far
   * forward the calendar can navigate.
   */
  const lastAvailableDate =
    futureDepartures.length > 0
      ? parseDateOnly(
          futureDepartures[
            futureDepartures.length - 1
          ].date,
        )
      : undefined;

  /*
   * A calendar date is selectable only when:
   *
   * 1. it is today or later
   * 2. fare data exists for that exact departure date
   */
  const isAvailableFutureDate = (date) => {
    const normalizedDate = new Date(date);

    normalizedDate.setHours(0, 0, 0, 0);

    return (
      normalizedDate >= today &&
      availableDateKeys.has(
        toDateKey(normalizedDate),
      )
    );
  };

  /*
   * ------------------------------------------------------------
   * LOAD DEPARTURE PRICES
   * ------------------------------------------------------------
   */
  useEffect(() => {
    if (!routeKey) return;

    let cancelled = false;

    axios
      .get(
        `${API}/api/prices?route=${encodeURIComponent(
          routeKey,
        )}&mode=depart`,
      )
      .then((res) => {
        if (cancelled) return;

        setDepartureResult({
          routeKey,
          data: cleanPriceData(res.data.prices),
          error: "",
        });
      })
      .catch(() => {
        if (cancelled) return;

        setDepartureResult({
          routeKey,
          data: [],
          error: "Could not load price data.",
        });
      });

    return () => {
      cancelled = true;
    };
  }, [routeKey]);

  /*
   * ------------------------------------------------------------
   * LOAD PRICE HISTORY
   * ------------------------------------------------------------
   */
  useEffect(() => {
    if (
      !routeKey ||
      mode !== "history" ||
      !selectedDepartDate
    ) {
      return;
    }

    let cancelled = false;

    const historyKey =
      `${routeKey}-${selectedDepartDate}`;

    axios
      .get(
        `${API}/api/prices?route=${encodeURIComponent(
          routeKey,
        )}&mode=history&departDate=${encodeURIComponent(
          selectedDepartDate,
        )}`,
      )
      .then((res) => {
        if (cancelled) return;

        setHistoryResult({
          key: historyKey,
          data: cleanPriceData(res.data.prices),
          error: "",
        });
      })
      .catch(() => {
        if (cancelled) return;

        setHistoryResult({
          key: historyKey,
          data: [],
          error: "Could not load price history.",
        });
      });

    return () => {
      cancelled = true;
    };
  }, [
    routeKey,
    mode,
    selectedDepartDate,
  ]);

  if (!route) return null;

  /*
   * ------------------------------------------------------------
   * DERIVED STATE
   * ------------------------------------------------------------
   */
  const historyKey = selectedDepartDate
    ? `${routeKey}-${selectedDepartDate}`
    : "";

  const departureLoading =
    mode === "depart" &&
    departureResult.routeKey !== routeKey;

  const historyLoading =
    mode === "history" &&
    Boolean(selectedDepartDate) &&
    historyResult.key !== historyKey;

  const loading =
    departureLoading || historyLoading;

  const chartData =
    mode === "depart"
      ? departures
      : historyResult.key === historyKey
        ? historyResult.data
        : [];

  const error =
    mode === "depart"
      ? departureResult.routeKey === routeKey
        ? departureResult.error
        : ""
      : historyResult.key === historyKey
        ? historyResult.error
        : "";

  /*
   * Keep latest 10 points for now.
   */
  const visibleHistory =
    chartData.slice(-10);

  const prices = visibleHistory.map(
    (item) => item.price,
  );

  const minPrice =
    prices.length > 0
      ? Math.min(...prices)
      : null;

  const avgPrice =
    prices.length > 0
      ? Math.round(
          prices.reduce(
            (a, b) => a + b,
            0,
          ) / prices.length,
        )
      : null;

  const maxPrice =
    prices.length > 0
      ? Math.max(...prices)
      : null;

  const hasPriceMovement =
    minPrice !== null &&
    maxPrice !== null &&
    minPrice !== maxPrice;

  /*
   * ------------------------------------------------------------
   * USER ACTIONS
   * ------------------------------------------------------------
   */

  /*
   * Clicking a departure-price point opens Price History
   * only if that departure date is today or later.
   */
  const openPriceHistory = (departDate) => {
    if (!departDate) return;

    const date = parseDateOnly(departDate);

    if (
      date < today ||
      !availableDateKeys.has(departDate)
    ) {
      return;
    }

    setView({
      routeKey,
      mode: "history",
      departDate,
    });
  };

  /*
   * Return to departure-price view.
   */
  const showDeparturePrices = () => {
    setView({
      routeKey,
      mode: "depart",
      departDate: selectedDepartDate,
    });
  };

  /*
   * Enter Price History mode.
   *
   * If an already selected date is still valid,
   * keep it.
   *
   * Otherwise select the nearest upcoming
   * departure date with fare data.
   */
  const showPriceHistory = () => {
    const selectedIsStillValid =
      selectedDepartDate &&
      availableDateKeys.has(
        selectedDepartDate,
      ) &&
      parseDateOnly(
        selectedDepartDate,
      ) >= today;

    const departDate =
      selectedIsStillValid
        ? selectedDepartDate
        : futureDepartures[0]?.date;

    if (!departDate) return;

    setView({
      routeKey,
      mode: "history",
      departDate,
    });
  };

  /*
   * Calendar selection.
   */
  const changeHistoryDeparture = (date) => {
    if (!date) return;

    const departDate = toDateKey(date);

    if (
      !isAvailableFutureDate(date)
    ) {
      return;
    }

    setView({
      routeKey,
      mode: "history",
      departDate,
    });
  };

  /*
   * ------------------------------------------------------------
   * CLICKABLE DEPARTURE DOTS
   * ------------------------------------------------------------
   */
  const renderDepartureDot = ({
    cx,
    cy,
    payload,
  }) => {
    if (
      cx == null ||
      cy == null ||
      !payload?.date
    ) {
      return null;
    }

    const isFuture =
      parseDateOnly(payload.date) >= today;

    return (
      <circle
        cx={cx}
        cy={cy}
        r={4}
        fill="#0891b2"
        stroke="#ffffff"
        strokeWidth={2}
        opacity={isFuture ? 1 : 0.35}
        style={{
          cursor: isFuture
            ? "pointer"
            : "default",
        }}
        onClick={() => {
          if (isFuture) {
            openPriceHistory(
              payload.date,
            );
          }
        }}
      />
    );
  };

  const renderActiveDepartureDot = ({
    cx,
    cy,
    payload,
  }) => {
    if (
      cx == null ||
      cy == null ||
      !payload?.date
    ) {
      return null;
    }

    const isFuture =
      parseDateOnly(payload.date) >= today;

    return (
      <circle
        cx={cx}
        cy={cy}
        r={isFuture ? 7 : 4}
        fill="#0e7490"
        stroke="#ffffff"
        strokeWidth={2}
        opacity={isFuture ? 1 : 0.35}
        style={{
          cursor: isFuture
            ? "pointer"
            : "default",
        }}
        onClick={() => {
          if (isFuture) {
            openPriceHistory(
              payload.date,
            );
          }
        }}
      />
    );
  };

  /*
   * ------------------------------------------------------------
   * BUTTON STYLING
   * ------------------------------------------------------------
   */
  const tabStyle = (
    active,
    disabled = false,
  ) => ({
    border: "none",
    borderRadius: 8,
    padding: "8px 12px",
    cursor: disabled
      ? "not-allowed"
      : "pointer",
    background: active
      ? "#0891b2"
      : "transparent",
    color: active
      ? "#fff"
      : disabled
        ? "#94a3b8"
        : "#334155",
    fontWeight: 600,
    opacity: disabled ? 0.6 : 1,
  });

  return (
    <div className="chart-wrapper">
      {/*
       * ----------------------------------------------------------
       * HEADER
       * ----------------------------------------------------------
       */}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "flex-start",
          gap: 12,
          flexWrap: "wrap",
          marginBottom: 12,
        }}
      >
        <h2
          className="chart-title"
          style={{ margin: 0 }}
        >
          {route.origin} →{" "}
          {route.destination}

          <span className="chart-subtitle">
            {DESTINATION_LABELS[
              route.destination
            ] || route.destination}

            {" · "}

            {mode === "depart"
              ? "price by departure date"
              : selectedDepartDate
                ? `price history for ${formatDate(
                    selectedDepartDate,
                    {
                      month: "short",
                      day: "numeric",
                      year: "numeric",
                    },
                  )}`
                : "price movement over time"}
          </span>
        </h2>

        <div
          style={{
            display: "inline-flex",
            background: "#f1f5f9",
            borderRadius: 10,
            padding: 4,
            gap: 4,
          }}
        >
          <button
            type="button"
            onClick={
              showDeparturePrices
            }
            style={tabStyle(
              mode === "depart",
            )}
          >
            Departure prices
          </button>

          <button
            type="button"
            disabled={
              futureDepartures.length === 0
            }
            onClick={showPriceHistory}
            style={tabStyle(
              mode === "history",
              futureDepartures.length === 0,
            )}
          >
            Price history
          </button>
        </div>
      </div>

      {/*
       * ----------------------------------------------------------
       * CALENDAR
       * ----------------------------------------------------------
       *
       * Opens on current month.
       * Cannot navigate before current month.
       * Dates before today are disabled.
       * Dates without fare data are disabled.
       */}
      {mode === "history" &&
        futureDepartures.length > 0 && (
          <div
            style={{
              marginBottom: 20,
              padding: 16,
              background: "#f8fafc",
              border:
                "1px solid #e2e8f0",
              borderRadius: 12,
              display: "inline-block",
            }}
          >
            <div
              style={{
                marginBottom: 4,
                fontSize: 13,
                fontWeight: 700,
                color: "#334155",
              }}
            >
              Select departure date
            </div>

            <div
              style={{
                marginBottom: 8,
                fontSize: 12,
                color: "#64748b",
              }}
            >
              Only upcoming dates with
              fare data can be selected.
            </div>

            <DayPicker
              mode="single"
              selected={
                selectedCalendarDate
              }
              onSelect={
                changeHistoryDeparture
              }

              /*
               * Always open at current month.
               */
              defaultMonth={
                currentMonth
              }

              /*
               * Prevent navigation to
               * months before current month.
               */
              startMonth={
                currentMonth
              }

              /*
               * Don't navigate beyond our
               * last future departure date.
               */
              endMonth={
                lastAvailableDate
              }

              /*
               * Disable past dates and
               * dates without fare data.
               */
              disabled={(date) =>
                !isAvailableFutureDate(
                  date,
                )
              }

              /*
               * Give available dates
               * a stronger appearance.
               */
              modifiers={{
                available: (date) =>
                  isAvailableFutureDate(
                    date,
                  ),
              }}

              modifiersStyles={{
                available: {
                  fontWeight: 700,
                },
              }}
            />

            {selectedDepartDate && (
              <div
                style={{
                  marginTop: 10,
                  paddingTop: 10,
                  borderTop:
                    "1px solid #e2e8f0",
                  fontSize: 12,
                  color: "#64748b",
                }}
              >
                Showing history for{" "}
                <strong
                  style={{
                    color: "#0f172a",
                  }}
                >
                  {formatDate(
                    selectedDepartDate,
                    {
                      weekday: "short",
                      month: "short",
                      day: "numeric",
                      year: "numeric",
                    },
                  )}
                </strong>
              </div>
            )}
          </div>
        )}

      {/*
       * ----------------------------------------------------------
       * CHART CONTENT
       * ----------------------------------------------------------
       */}
      {loading ? (
        <div className="chart-empty">
          Loading...
        </div>
      ) : error ? (
        <div className="chart-empty">
          {error}
        </div>
      ) : chartData.length === 0 ? (
        <div className="chart-empty">
          {mode === "history"
            ? "No price history yet for this departure date."
            : "No price data yet — search a route above to start building history."}
        </div>
      ) : (
        <>
          {/*
           * ------------------------------------------------------
           * PRICE STATS
           * ------------------------------------------------------
           */}
          <div className="chart-stats">
            <div className="chart-stat">
              <span className="chart-stat-label">
                Lowest
              </span>

              <span className="chart-stat-value lowest">
                $
                {minPrice.toLocaleString()}
              </span>
            </div>

            <div className="chart-stat">
              <span className="chart-stat-label">
                Average
              </span>

              <span className="chart-stat-value">
                $
                {avgPrice.toLocaleString()}
              </span>
            </div>
          </div>

          {/*
           * ------------------------------------------------------
           * PRICE CHART
           * ------------------------------------------------------
           */}
          <ResponsiveContainer
            width="100%"
            height={240}
          >
            <LineChart
              data={visibleHistory}
              margin={{
                top: 8,
                right: 16,
                left: 0,
                bottom: 0,
              }}
            >
              <CartesianGrid
                strokeDasharray="3 3"
                stroke="rgba(0,0,0,0.06)"
              />

              <XAxis
                dataKey="date"
                tick={{
                  fontSize: 10,
                  fill: "#888",
                }}
                tickFormatter={(d) =>
                  formatDate(d, {
                    month: "short",
                    day: "numeric",
                  })
                }
              />

              <YAxis
                tick={{
                  fontSize: 11,
                  fill: "#888",
                }}
                tickFormatter={(v) =>
                  `$${v}`
                }
                domain={[
                  "auto",
                  "auto",
                ]}
              />

              <Tooltip
                contentStyle={{
                  background: "#fff",
                  border:
                    "1px solid #e2e8f0",
                  borderRadius: 8,
                  boxShadow:
                    "0 4px 16px rgba(0,0,0,0.08)",
                }}
                formatter={(v) => [
                  `$${Number(
                    v,
                  ).toLocaleString()}`,
                  "Price",
                ]}
                labelFormatter={(d) =>
                  mode === "depart"
                    ? `Departing ${formatDate(
                        d,
                        {
                          month: "short",
                          day: "numeric",
                          year: "numeric",
                        },
                      )}`
                    : `Checked ${formatDate(
                        d,
                        {
                          month: "short",
                          day: "numeric",
                          year: "numeric",
                        },
                      )}`
                }
                labelStyle={{
                  color: "#64748b",
                  fontSize: 12,
                }}
              />

              {hasPriceMovement && (
                <ReferenceLine
                  y={avgPrice}
                  stroke="#94a3b8"
                  strokeDasharray="4 4"
                  label={{
                    value: "avg",
                    position: "right",
                    fontSize: 10,
                    fill: "#94a3b8",
                  }}
                />
              )}

              <Line
                type="monotone"
                dataKey="price"
                stroke="#0891b2"
                strokeWidth={2.5}
                dot={
                  mode === "depart"
                    ? renderDepartureDot
                    : false
                }
                activeDot={
                  mode === "depart"
                    ? renderActiveDepartureDot
                    : {
                        r: 6,
                        fill: "#0e7490",
                      }
                }
              />
            </LineChart>
          </ResponsiveContainer>
        </>
      )}
    </div>
  );
}